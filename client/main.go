package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("expected client number as first argument")
	}
	n := os.Args[1]
	fmt.Printf("Hello, client %v.\n", n)
}
