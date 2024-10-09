package main

import (
	"distribuidos/tp1/server/middleware"
	"fmt"
)

const Q1 = 3

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
	generateNet()
}

func generateInit() {
	fmt.Println("name: tp1")
	fmt.Println("services:")
	fmt.Println("  services:")
}

func generateRabbit() {
	fmt.Println("    rabbitmq:")
	fmt.Println("      container_name: rabbitmq")
	fmt.Println("      image: rabbitmq:4-management")
	fmt.Println("      ports:")
	fmt.Println("        - 5672:5672")
	fmt.Println("        - 15672:15672")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      healthcheck:")
	fmt.Println("        test: [\"CMD\", \"rabbitmqctl\", \"status\"]")
	fmt.Println("        interval: 5s")
	fmt.Println("        timeout: 5s")
	fmt.Println("        retries: 3")
}

func generateGateway() {
	fmt.Println("    gateway:")
	fmt.Println("      container_name: gateway")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /gateway")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        rabbitmq:")
	fmt.Println("          condition: service_healthy")
}

func generateClient() {
	fmt.Println("    client:")
	fmt.Println("      container_name: client")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /client")
	fmt.Println("      environment:")
	fmt.Println("        - GATEWAY_CONN_ADDR=gateway:9001")
	fmt.Println("        - GATEWAY_DATA_ADDR=gateway:9002")
	fmt.Println("      volumes:")
	fmt.Println("        - ./.data:/.data")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
}

func generateGenreFilter() {
	fmt.Println("    genre-filter:")
	fmt.Println("      container_name: genre-filter")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /genre-filter")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
}

func generateDecadeFilter() {
	fmt.Println("    decade-filter:")
	fmt.Println("      container_name: decade-filter")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /decade-filter")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Println("        - DECADE=2010")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
}

func generateScoreFilter() {
	fmt.Println("    review-filter:")
	fmt.Println("      container_name: review-filter")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /review-filter")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
}

func generateLanguageFilter() {
	fmt.Println("    language-filter:")
	fmt.Println("      container_name: language-filter")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /language-filter")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
}

func generateQ1() {
	fmt.Println("    q1-partitioner:")
	fmt.Println("      container_name: q1-partitioner")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /partitioner")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Printf("        - INPUT=%v\n", middleware.GamesPerPlatformQueue)
	fmt.Printf("        - PARTITIONS=%v\n", Q1)
	fmt.Println("        - TYPE=game")
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
	for i := 1; i <= Q1; i++ {
		fmt.Printf("    q1-%v:\n", i)
		fmt.Printf("      container_name: q1-%v:\n", i)
		fmt.Println("      image: tp1:latest")
		fmt.Println("      entrypoint: /games-per-platform")
		fmt.Println("      environment:")
		fmt.Println("        - RABBIT_IP=rabbitmq")
		fmt.Printf("        - PARTITION_ID=%v\n", i)
		fmt.Println("      networks:")
		fmt.Println("        - net")
		fmt.Println("      depends_on:")
		fmt.Println("        - gateway")
	}
	fmt.Println("    q1-joiner:")
	fmt.Println("      container_name: q1-joiner")
	fmt.Println("      image: tp1:latest")
	fmt.Println("      entrypoint: /games-per-platform-joiner")
	fmt.Println("      environment:")
	fmt.Println("        - RABBIT_IP=rabbitmq")
	fmt.Printf("        - PARTITIONS=%v\n", Q1)
	fmt.Println("      networks:")
	fmt.Println("        - net")
	fmt.Println("      depends_on:")
	fmt.Println("        - gateway")
}

func generateNet() {
	fmt.Println("networks:")
	fmt.Println("  net:")
}
