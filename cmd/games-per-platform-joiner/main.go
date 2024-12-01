package main

import (
	"context"
	"distribuidos/tp1/database"
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/protocol"
	"distribuidos/tp1/utils"
	"encoding/binary"
	"encoding/gob"
	"io"
	"os/signal"
	"syscall"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

var log = logging.MustGetLogger("log")

type config struct {
	RabbitIP   string
	Partitions int
}

func getConfig() (config, error) {
	v := viper.New()

	v.SetDefault("RabbitIP", "localhost")
	v.SetDefault("Partitions", "1")

	_ = v.BindEnv("RabbitIP", "RABBIT_IP")
	_ = v.BindEnv("Partitions", "PARTITIONS")

	var c config
	err := v.Unmarshal(&c)

	return c, err
}

type Platform string

const (
	Mac     Platform = "mac"
	Linux   Platform = "linux"
	Windows Platform = "windows"
)

type handler struct {
	db     *database.Database
	output string
	joiner *middleware.JoinerDisk
}

func buildHandler(partition int) middleware.HandlerFunc[*handler] {
	return func(h *handler, ch *middleware.Channel, data []byte) error {
		return h.handlePartialResult(ch, data, partition)
	}
}

func (h *handler) handlePartialResult(ch *middleware.Channel, data []byte, partition int) error {
	snapshot, err := h.db.NewSnapshot()
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			cerr := snapshot.Commit()
			utils.Expect(cerr, "unrecoverable error")
		default:
			cerr := snapshot.Abort()
			utils.Expect(cerr, "unrecoverable error")
		}
	}()

	c, err := middleware.Deserialize[map[Platform]int](data)
	if h.joiner.Seen(partition) {
		if h.joiner.EOF() {
			ch.Finish()
		}
		return nil
	}
	err = h.joiner.Mark(snapshot, partition)
	if err != nil {
		return err
	}

	var windowsCounter uint64
	var linuxCounter uint64
	var macCounter uint64

	exists, err := snapshot.Exists("counter")
	if err != nil {
		return err
	}
	counterFile, err := snapshot.Update("counter")
	if err != nil {
		return err
	}
	if exists {
		err = binary.Read(counterFile, binary.LittleEndian, &windowsCounter)
		if err != nil {
			return err
		}
		err = binary.Read(counterFile, binary.LittleEndian, &linuxCounter)
		if err != nil {
			return err
		}
		err = binary.Read(counterFile, binary.LittleEndian, &macCounter)
		if err != nil {
			return err
		}
	}

	windowsCounter += uint64(c[Windows])
	linuxCounter += uint64(c[Linux])
	macCounter += uint64(c[Mac])

	_, err = counterFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	err = binary.Write(counterFile, binary.LittleEndian, windowsCounter)
	if err != nil {
		return err
	}
	err = binary.Write(counterFile, binary.LittleEndian, linuxCounter)
	if err != nil {
		return err
	}
	err = binary.Write(counterFile, binary.LittleEndian, macCounter)
	if err != nil {
		return err
	}

	utils.MaybeExit(0.001)

	if h.joiner.EOF() {
		count := map[Platform]int{
			Windows: int(windowsCounter),
			Linux:   int(linuxCounter),
			Mac:     int(macCounter),
		}
		for k, v := range count {
			log.Infof("Found %v games with %v support", v, string(k))
		}

		result := protocol.Q1Result{
			Windows: int(windowsCounter),
			Linux:   int(linuxCounter),
			Mac:     int(macCounter),
		}
		err := ch.SendAny(result, "", h.output)
		if err != nil {
			return err
		}

		utils.MaybeExit(0.50)

		ch.Finish()
		return nil
	}

	return nil
}

func (h *handler) Free() error {
	return h.db.Delete()
}

func main() {
	cfg, err := getConfig()
	utils.Expect(err, "Failed to read config")
	gob.Register(protocol.Q1Result{})

	conn, ch, err := middleware.Dial(cfg.RabbitIP)
	utils.Expect(err, "Failed to dial rabbit")

	queues := make([]middleware.QueueConfig, 0)
	endpoints := make(map[string]middleware.HandlerFunc[*handler], 0)

	for i := 1; i <= cfg.Partitions; i++ {
		qName := middleware.Cat(middleware.PartialQ1, i)
		qcfg := middleware.QueueConfig{
			Name: qName,
		}
		queues = append(queues, qcfg)
		endpoints[qName] = buildHandler(i)
	}

	queues = append(queues, middleware.QueueConfig{Name: middleware.Results})

	err = middleware.Topology{
		Queues: queues,
	}.Declare(ch)

	if err != nil {
		utils.Expect(err, "Failed to initialize topology")
	}

	nConfig := middleware.Config[*handler]{
		Builder: func(clientID int) *handler {
			database_path := middleware.Cat("client", clientID)
			db, err := database.NewDatabase(database_path)
			utils.Expect(err, "unrecoverable error")

			joiner := middleware.NewJoinerDisk("joiner", cfg.Partitions)
			err = joiner.Load(db)
			utils.Expect(err, "unrecoverable error")

			return &handler{
				db:     db,
				output: middleware.Results,
				joiner: joiner,
			}
		},
		Endpoints: endpoints,
	}

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	node, err := middleware.NewNode(nConfig, conn)
	if err != nil {
		utils.Expect(err, "Failed to start joiner node")
	}

	err = node.Run(ctx)
	if err != nil {
		utils.Expect(err, "Failed to run joiner node")
	}
}
