package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
)

const Q1 = 3
const Q2 = 3
const Q3 = 3
const Q4 = 3
const Q5 = 3

const LANGUAGE_FILTER = 3
const DECADE_FILTER = 3
const GENRE_FITLER = 3
const SCORE_FILTER = 3

func main() {
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
	generateNet()
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
	fmt.Println("    entrypoint: /gateway")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      rabbitmq:")
	fmt.Println("        condition: service_healthy")
}

func generateClient() {
	fmt.Println("  client:")
	fmt.Println("    container_name: client")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /client")
	fmt.Println("    environment:")
	fmt.Println("      - GATEWAY_CONN_ADDR=gateway:9001")
	fmt.Println("      - GATEWAY_DATA_ADDR=gateway:9002")
	fmt.Println("    volumes:")
	fmt.Println("      - ./.data:/.data")
	fmt.Println("      - ./.results:/.results")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
}

func generateGenreFilter() {
	fmt.Println("  genre-filter:")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /genre-filter")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	fmt.Println("    deploy:")
	fmt.Println("      mode: replicated")
	fmt.Printf("      replicas: %v\n", GENRE_FITLER)
}

func generateDecadeFilter() {
	fmt.Println("  decade-filter:")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /decade-filter")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - DECADE=2010")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	fmt.Println("    deploy:")
	fmt.Println("      mode: replicated")
	fmt.Printf("      replicas: %v\n", DECADE_FILTER)
}

func generateScoreFilter() {
	fmt.Println("  review-filter:")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /review-filter")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	fmt.Println("    deploy:")
	fmt.Println("      mode: replicated")
	fmt.Printf("      replicas: %v\n", SCORE_FILTER)
}

func generateLanguageFilter() {
	fmt.Println("  language-filter:")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /language-filter")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	fmt.Println("    deploy:")
	fmt.Println("      mode: replicated")
	fmt.Printf("      replicas: %v\n", LANGUAGE_FILTER)
}

func generateQ1() {
	fmt.Println("  q1-partitioner:")
	fmt.Println("    container_name: q1-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /partitioner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - INPUT=%v\n", middleware.GamesQ1)
	fmt.Printf("      - PARTITIONS=%v\n", Q1)
	fmt.Println("      - TYPE=game")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
	for i := 1; i <= Q1; i++ {
		fmt.Printf("  q1-count-%v:\n", i)
		fmt.Printf("    container_name: q1-count-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /games-per-platform")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
	}
	fmt.Println("  q1-joiner:")
	fmt.Println("    container_name: q1-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /games-per-platform-joiner")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Printf("      - PARTITIONS=%v\n", Q1)
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
}

func generateQ2() {
	fmt.Println("  q2-partitioner:")
	fmt.Println("    container_name: q2-partitioner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /partitioner")
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
		fmt.Println("    entrypoint: /top-n-historic-avg")
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
	fmt.Println("    entrypoint: /top-n-historic-avg-joiner")
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
	fmt.Println("    entrypoint: /partitioner")
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
	fmt.Println("    entrypoint: /partitioner")
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
		fmt.Println("    entrypoint: /group-by-game")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - GAME_INPUT=%v\n", middleware.GamesQ3)
		fmt.Printf("      - REVIEW_INPUT=%v\n", middleware.ReviewsQ3)
		fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ3)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")

		fmt.Printf("  q3-top-%v:\n", i)
		fmt.Printf("    container_name: q3-top-%v\n", i)
		fmt.Println("    image: tp1:latest")
		fmt.Println("    entrypoint: /top-n-reviews")
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
	fmt.Println("    entrypoint: /top-n-reviews-joiner")
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
	fmt.Println("    entrypoint: /partitioner")
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
	fmt.Println("    entrypoint: /partitioner")
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
		fmt.Println("    entrypoint: /group-by-game")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - GAME_INPUT=%v\n", middleware.GamesQ4)
		fmt.Printf("      - REVIEW_INPUT=%v\n", middleware.ReviewsQ4)
		fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ4Joiner)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
	}

	fmt.Println("  q4-joiner:")
	fmt.Println("    container_name: q4-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /group-joiner")
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
	fmt.Println("    entrypoint: /more-than-n-reviews")
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
	fmt.Println("    entrypoint: /partitioner")
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
	fmt.Println("    entrypoint: /partitioner")
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
		fmt.Println("    entrypoint: /group-by-game")
		fmt.Println("    environment:")
		fmt.Println("      - RABBIT_IP=rabbitmq")
		fmt.Printf("      - PARTITION_ID=%v\n", i)
		fmt.Printf("      - GAME_INPUT=%v\n", middleware.GamesQ5)
		fmt.Printf("      - REVIEW_INPUT=%v\n", middleware.ReviewsQ5)
		fmt.Printf("      - OUTPUT=%v\n", middleware.GroupedQ5Joiner)
		fmt.Println("    networks:")
		fmt.Println("      - net")
		fmt.Println("    depends_on:")
		fmt.Println("      - gateway")
	}

	fmt.Println("  q5-joiner:")
	fmt.Println("    container_name: q5-joiner")
	fmt.Println("    image: tp1:latest")
	fmt.Println("    entrypoint: /group-joiner")
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
	fmt.Println("    entrypoint: /90-percentile")
	fmt.Println("    environment:")
	fmt.Println("      - RABBIT_IP=rabbitmq")
	fmt.Println("      - PERCENTILE=90")
	fmt.Println("    networks:")
	fmt.Println("      - net")
	fmt.Println("    depends_on:")
	fmt.Println("      - gateway")
}

func generateNet() {
	fmt.Println("networks:")
	fmt.Println("  net:")
}
