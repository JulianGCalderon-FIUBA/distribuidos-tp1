package main

import (
	"distribuidos/tp1/middleware"
	"encoding/csv"
	"fmt"
	"os"

	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

const CLIENT = 1

const Q1 = 1
const Q2 = 1
const Q3 = 1
const Q4 = 1
const Q5 = 1

const LANGUAGE_FILTER = 1
const DECADE_FILTER = 1
const GENRE_FITLER = 1
const SCORE_FILTER = 1

const RESTARTER = 1

type Container struct {
	Name string
	Port string
}

var containers []Container

func main() {
	generateInit()
	generateRabbit()
	generateGateway()
	// generateClient()
	generateGenreFilter()
	generateDecadeFilter()
	generateScoreFilter()
	generateLanguageFilter()
	generateQ1()
	// generateQ2()
	// generateQ3()
	// generateQ4()
	// generateQ5()
	generateRestarter()
	generateNet()

	writeNodeConfig(".node-config.csv")
}

func addNodeConfig(name, port string) {
	containers = append(containers, Container{Name: name, Port: port})
}

func writeNodeConfig(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Errorf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write([]string{"name", "port"})
	if err != nil {
		log.Errorf("Error writing file: %v", err)
		return
	}

	for _, container := range containers {
		err = writer.Write([]string{container.Name, container.Port})
		if err != nil {
			log.Errorf("Error writing file: %v", err)
			return
		}

	}
	log.Infof("Nodes configuration written to %s\n", filename)
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
	fmt.Println("    logging:")
	fmt.Println("      driver: none")
}

func generateGateway() {
	fmt.Println("  gateway:")
	fmt.Println("    container_name: gateway")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/gateway")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      rabbitmq:")
	fmt.Println("        condition: service_healthy")
	// addNodeConfig("gateway", "7000")
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
		fmt.Println("      - ./.data:/work/.data")
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
		// fmt.Println("    deploy:")
		// fmt.Println("      mode: replicated")
		// fmt.Printf("      replicas: %v\n", GENRE_FITLER)
		addNodeConfig(fmt.Sprintf("genre-filter-%v", i), "7000")
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
		fmt.Printf("      - ADDRESS=decade-filter-%v:7000\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		// fmt.Println("    deploy:")
		// fmt.Println("      mode: replicated")
		// fmt.Printf("      replicas: %v\n", DECADE_FILTER)
		addNodeConfig(fmt.Sprintf("decade-filter-%v", i), "7000")
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
		fmt.Printf("      - ADDRESS=review-filter-%v:7000\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		// fmt.Println("    deploy:")
		// fmt.Println("      mode: replicated")
		// fmt.Printf("      replicas: %v\n", SCORE_FILTER)
		addNodeConfig(fmt.Sprintf("review-filter-%v", i), "7000")
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
		fmt.Printf("      - ADDRESS=language-filter-%v:7000\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		// fmt.Println("    deploy:")
		// fmt.Println("      mode: replicated")
		// fmt.Printf("      replicas: %v\n", LANGUAGE_FILTER)
		addNodeConfig(fmt.Sprintf("language-filter-%v", i), "7000")
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
	fmt.Println("      - ADDRESS=q1-partitioner:7000")
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q1-partitioner", "7000")
	for i := 1; i <= Q1; i++ {
		fmt.Printf("  q1-count-%v:\n", i)
		fmt.Printf("    container_name: q1-count-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/games-per-platform")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - ADDRESS=q1-count-%v:7000\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
		addNodeConfig(fmt.Sprintf("q1-count-%v", i), "7000")
	}
	fmt.Println("  q1-joiner:")
	fmt.Println("    container_name: q1-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/games-per-platform-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - PARTITIONS=%v\n", Q1)
	fmt.Println("      - ADDRESS=q1-joiner:7000")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	addNodeConfig("q1-joiner", "7000")
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
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
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
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
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

	fmt.Println("  q3-reviews-partitioner:")
	fmt.Println("    container_name: q3-reviews-partitioner")
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
		fmt.Println("    volumes:")
		fmt.Printf("      - ./.backup/q3-group-%v:/work\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")

		fmt.Printf("  q3-top-%v:\n", i)
		fmt.Printf("    container_name: q3-top-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/top-n-reviews")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Println("      - N=5000")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
	}
	fmt.Println("  q3-joiner:")
	fmt.Println("    container_name: q3-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/top-n-reviews-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - TOP_N=5")
	fmt.Printf("      - PARTITIONS=%v\n", Q3)
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
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

	fmt.Println("  q4-reviews-partitioner:")
	fmt.Println("    container_name: q4-reviews-partitioner")
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
		fmt.Println("    volumes:")
		fmt.Printf("      - ./.backup/q4-group-%v:/work\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
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
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")

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

	fmt.Println("  q5-reviews-partitioner:")
	fmt.Println("    container_name: q5-reviews-partitioner")
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
		fmt.Println("    volumes:")
		fmt.Printf("      - ./.backup/q5-group-%v:/work\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
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
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")

	fmt.Println("  q5-percentile:")
	fmt.Println("    container_name: q5-percentile")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /build/percentile")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - PERCENTILE=90")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
}

func generateRestarter() {
	for i := 0; i < RESTARTER; i++ {
		fmt.Printf("  restarter-%v:\n", i)
		fmt.Printf("    container_name: restarter-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /build/restarter")
		fmt.Println("    environment:")
		fmt.Printf("      - ID=%v\n", i)
		fmt.Printf("      - ADDRESS=restarter-%v:7000\n", i)
		fmt.Printf("      - REPLICAS=%v\n", RESTARTER)
		fmt.Println("    volumes:")
		fmt.Println("      - ./.node-config.csv:/work/.node-config.csv")
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
	}
}

func generateNet() {
	fmt.Println("networks:")
	fmt.Println("  net:")
}
