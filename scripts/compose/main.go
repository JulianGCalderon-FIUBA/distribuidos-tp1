package main

import (
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"flag"
	"fmt"
	"os"
	"strings"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

const CLIENT = 3

const Q1 = 3
const Q2 = 3
const Q3 = 3
const Q4 = 3
const Q5 = 3

const LANGUAGE_FILTER = 4
const DECADE_FILTER = 3
const GENRE_FITLER = 3
const SCORE_FILTER = 4
const PARTITIONERS = 3

const RESTARTER = 4

var names []string

var volumes bool

func main() {
	flag.BoolVar(&volumes, "volumes", true, "setup bind mounts")
	flag.Parse()

	generateInit()
	generateRabbit()
	generateGateway()
	generateClient()
	generateGenreFilter()
	generateDecadeFilter()
	generateScoreFilter()
	generateLanguageFilter()
	generateQ1()
	generateQ2()
	generateQ3()
	generateQ4()
	generateQ5()
	generateRestarter()
	generateKiller()
	generateNet()

	writeRestarterConfig(".restarter-config")
	writeKillerConfig(".killer-config")
}

func addNodeConfig(name string) {
	names = append(names, name)
}

func writeRestarterConfig(filename string) {
	file, err := os.Create(filename)
	utils.Expect(err, "Failed to create restarter config")
	defer file.Close()

	for _, name := range names {
		if strings.Contains(name, "restarter") {
			continue
		}

		_, err = file.WriteString(name)
		utils.Expect(err, "Failed to write restarter config")

		_, err = file.Write([]byte{'\n'})
		utils.Expect(err, "Failed to write restarter config")
	}

	log.Infof("Restarter configuration written to %s\n", filename)
}

func writeKillerConfig(filename string) {
	file, err := os.Create(filename)
	utils.Expect(err, "Failed to create killer config")
	defer file.Close()

	for _, name := range names {
		if name == "restarter-0" {
			continue
		}
		if name == "gateway" {
			continue
		}

		_, err = file.WriteString(name)
		utils.Expect(err, "failed to write killer config")

		_, err = file.Write([]byte{'\n'})
		utils.Expect(err, "failed to write killer config")
	}

	log.Infof("Killer configuration written to %s\n", filename)
}

func generateInit() {
	fmt.Println("name: tp1")
	fmt.Println("services:")
}

func generateRabbit() {
	fmt.Println("  rabbitmq:")
	fmt.Println("    container_name: rabbitmq")
	fmt.Println("    image: rabbitmq:4-management")
	fmt.Println("    ports:")
	fmt.Println("      - 5672:5672")
	fmt.Println("      - 15672:15672")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    healthcheck:")
	fmt.Println("      test: [\"CMD\", \"rabbitmqctl\", \"status\"]")
	fmt.Println("      interval: 5s")
	fmt.Println("      timeout: 5s")
	fmt.Println("      retries: 3")
}

func generateGateway() {
	fmt.Println("  gateway:")
	fmt.Println("    container_name: gateway")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/gateway")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/gateway:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      rabbitmq:")
	fmt.Println("        condition: service_healthy")
	addNodeConfig("gateway")
}

func generateClient() {
	for i := 1; i <= CLIENT; i++ {
		fmt.Printf("  client-%v:\n", i)
		fmt.Printf("    container_name: client-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/client")
		fmt.Println("    environment:")
		fmt.Println("      - GATEWAY_CONN_ADDR=gateway:9001")
		fmt.Println("      - GATEWAY_DATA_ADDR=gateway:9002")
		fmt.Println("    volumes:")
		fmt.Println("      - ./.data-reduced:/work/.data")
		fmt.Printf("      - ./.results-%v:/work/.results\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
	}
}

func generateGenreFilter() {
	for i := 1; i <= GENRE_FITLER; i++ {
		fmt.Printf("  genre-filter-%v:\n", i)
		fmt.Printf("    container_name: genre-filter-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/filter-genre")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - ADDRESS=genre-filter-%v:7000\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("genre-filter-%v", i))
	}
}

func generateDecadeFilter() {
	for i := 1; i <= DECADE_FILTER; i++ {
		fmt.Printf("  decade-filter-%v:\n", i)
		fmt.Printf("    container_name: decade-filter-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/filter-decade")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Println("      - DECADE=2010")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("decade-filter-%v", i))
	}
}

func generateScoreFilter() {
	for i := 1; i <= SCORE_FILTER; i++ {
		fmt.Printf("  review-filter-%v:\n", i)
		fmt.Printf("    container_name: review-filter-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/filter-score")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("review-filter-%v", i))
	}
}

func generateLanguageFilter() {
	for i := 1; i <= LANGUAGE_FILTER; i++ {
		fmt.Printf("  language-filter-%v:\n", i)
		fmt.Printf("    container_name: language-filter-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/filter-language")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("language-filter-%v", i))
	}
}

func generateQ1() {
	fmt.Println("  q1-partitioner:")
	fmt.Println("    container_name: q1-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/partitioner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - INPUT=%v\n", middleware.GamesQ1)
	fmt.Printf("      - PARTITIONS=%v\n", Q1)
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q1-partitioner")
	for i := 1; i <= Q1; i++ {
		fmt.Printf("  q1-count-%v:\n", i)
		fmt.Printf("    container_name: q1-count-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/games-per-platform")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		if volumes {
			fmt.Println("    volumes:")
			fmt.Printf("      - ./.backup/q1-count-%v:/work\n", i)
		}
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q1-count-%v", i))
	}
	fmt.Println("  q1-joiner:")
	fmt.Println("    container_name: q1-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/games-per-platform-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - PARTITIONS=%v\n", Q1)
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/q1-joiner:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q1-joiner")
}

func generateQ2() {
	fmt.Println("  q2-partitioner:")
	fmt.Println("    container_name: q2-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/partitioner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - INPUT=%v\n", middleware.GamesQ2)
	fmt.Printf("      - PARTITIONS=%v\n", Q2)
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q2-partitioner")
	for i := 1; i <= Q2; i++ {
		fmt.Printf("  q2-top-%v:\n", i)
		fmt.Printf("    container_name: q2-top-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/top-n-historic-avg")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - INPUT=%v\n", middleware.GamesQ2)
		fmt.Println("      - TOP_N=10")
		if volumes {
			fmt.Println("    volumes:")
			fmt.Printf("      - ./.backup/q2-top-%v:/work\n", i)
		}
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q2-top-%v", i))
	}
	fmt.Println("  q2-joiner:")
	fmt.Println("    container_name: q2-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/top-n-historic-avg-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - PARTITIONS=%v\n", Q2)
	fmt.Printf("      - INPUT=%v\n", middleware.PartialQ2)
	fmt.Println("      - TOP_N=10")
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/q2-joiner:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q2-joiner")
}

func generateQ3() {
	fmt.Println("  q3-games-partitioner:")
	fmt.Println("    container_name: q3-games-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/partitioner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - INPUT=%v\n", middleware.GamesQ3)
	fmt.Printf("      - PARTITIONS=%v\n", Q3)
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q3-games-partitioner")

	for i := 1; i <= PARTITIONERS; i++ {
		fmt.Printf("  q3-reviews-partitioner-%v:\n", i)
		fmt.Printf("    container_name: q3-reviews-partitioner-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/partitioner")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - INPUT=%v\n", middleware.ReviewsQ3)
		fmt.Printf("      - PARTITIONS=%v\n", Q3)
		fmt.Println("      - TYPE=review")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q3-reviews-partitioner-%v", i))
	}

	for i := 1; i <= Q3; i++ {
		fmt.Printf("  q3-group-%v:\n", i)
		fmt.Printf("    container_name: q3-group-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/group-by")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - GAME_INPUT=%v\n", middleware.GamesQ3)
		fmt.Printf("      - REVIEW_INPUT=%v\n", middleware.ReviewsQ3)
		fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ3)
		if volumes {
			fmt.Println("    volumes:")
			fmt.Printf("      - ./.backup/q3-group-%v:/work\n", i)
		}
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q3-group-%v", i))

		fmt.Printf("  q3-top-%v:\n", i)
		fmt.Printf("    container_name: q3-top-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/top-n-reviews")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Println("      - N=5")
		if volumes {
			fmt.Println("    volumes:")
			fmt.Printf("      - ./.backup/q3-top-%v:/work\n", i)
		}
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q3-top-%v", i))
	}
	fmt.Println("  q3-joiner:")
	fmt.Println("    container_name: q3-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/top-n-reviews-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - TOP_N=5")
	fmt.Printf("      - PARTITIONS=%v\n", Q3)
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/q3-joiner:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q3-joiner")
}

func generateQ4() {
	fmt.Println("  q4-games-partitioner:")
	fmt.Println("    container_name: q4-games-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/partitioner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - INPUT=%v\n", middleware.GamesQ4)
	fmt.Printf("      - PARTITIONS=%v\n", Q4)
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q4-games-partitioner")

	for i := 1; i <= PARTITIONERS; i++ {
		fmt.Printf("  q4-reviews-partitioner-%v:\n", i)
		fmt.Printf("    container_name: q4-reviews-partitioner-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/partitioner")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - INPUT=%v\n", middleware.ReviewsQ4)
		fmt.Printf("      - PARTITIONS=%v\n", Q4)
		fmt.Println("      - TYPE=review")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q4-reviews-partitioner-%v", i))
	}

	for i := 1; i <= Q4; i++ {
		fmt.Printf("  q4-group-%v:\n", i)
		fmt.Printf("    container_name: q4-group-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/group-by")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - GAME_INPUT=%v\n", middleware.GamesQ4)
		fmt.Printf("      - REVIEW_INPUT=%v\n", middleware.ReviewsQ4)
		fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ4Joiner)
		if volumes {
			fmt.Println("    volumes:")
			fmt.Printf("      - ./.backup/q4-group-%v:/work\n", i)
		}
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q4-group-%v", i))
	}

	fmt.Println("  q4-joiner:")
	fmt.Println("    container_name: q4-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/group-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - PARTITIONS=%v\n", Q4)
	fmt.Printf("      - INPUT=%v\n", middleware.GroupedQ4Joiner)
	fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ4Filter)
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/q4-joiner:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q4-joiner")

	fmt.Println("  q4-filter:")
	fmt.Println("    container_name: q4-filter")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/more-than-n-reviews")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - N=5000")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q4-filter")
}

func generateQ5() {
	fmt.Println("  q5-games-partitioner:")
	fmt.Println("    container_name: q5-games-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/partitioner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - INPUT=%v\n", middleware.GamesQ5)
	fmt.Printf("      - PARTITIONS=%v\n", Q5)
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q5-games-partitioner")

	for i := 1; i <= PARTITIONERS; i++ {
		fmt.Printf("  q5-reviews-partitioner-%v:\n", i)
		fmt.Printf("    container_name: q5-reviews-partitioner-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/partitioner")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - INPUT=%v\n", middleware.ReviewsQ5)
		fmt.Printf("      - PARTITIONS=%v\n", Q5)
		fmt.Println("      - TYPE=review")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q5-reviews-partitioner-%v", i))
	}

	for i := 1; i <= Q5; i++ {
		fmt.Printf("  q5-group-%v:\n", i)
		fmt.Printf("    container_name: q5-group-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/group-by")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - GAME_INPUT=%v\n", middleware.GamesQ5)
		fmt.Printf("      - REVIEW_INPUT=%v\n", middleware.ReviewsQ5)
		fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ5Joiner)
		if volumes {
			fmt.Println("    volumes:")
			fmt.Printf("      - ./.backup/q5-group-%v:/work\n", i)
		}
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q5-group-%v", i))
	}

	fmt.Println("  q5-joiner:")
	fmt.Println("    container_name: q5-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/group-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - PARTITIONS=%v\n", Q5)
	fmt.Printf("      - INPUT=%v\n", middleware.GroupedQ5Joiner)
	fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ5Percentile)
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/q5-joiner:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q5-joiner")

	fmt.Println("  q5-percentile:")
	fmt.Println("    container_name: q5-percentile")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/percentile")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - PERCENTILE=90")
	if volumes {
		fmt.Println("    volumes:")
		fmt.Println("      - ./.backup/q5-percentile:/work")
	}
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q5-percentile")
}

func generateRestarter() {
	for i := 0; i < RESTARTER; i++ {
		fmt.Printf("  restarter-%v:\n", i)
		fmt.Printf("    container_name: restarter-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/restarter")
		fmt.Println("    environment:")
		fmt.Printf("      - ID=%v\n", i)
		fmt.Printf("      - ADDRESS=restarter-%v:14300\n", i)
		fmt.Printf("      - REPLICAS=%v\n", RESTARTER)
		fmt.Println("    volumes:")
		fmt.Println("      - ./.restarter-config:/work/.restarter-config")
		fmt.Println("      - /var/run/docker.sock:/var/run/docker.sock")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		addNodeConfig(fmt.Sprintf("restarter-%v", i))
	}
}

func generateKiller() {
	fmt.Println("  killer:")
	fmt.Println("    container_name: killer")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/killer")
	fmt.Println("    environment:")
	fmt.Println("      - NODES_PATH=.killer-config")
	fmt.Println("      - PERIOD=5000")
	fmt.Println("    volumes:")
	fmt.Println("      - ./.killer-config:/work/.killer-config")
	fmt.Println("      - /var/run/docker.sock:/var/run/docker.sock")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	fmt.Println("    networks:")
	fmt.Println("      - net")
}

func generateNet() {
	fmt.Println("networks:")
	fmt.Println("  net:")
}
