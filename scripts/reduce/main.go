package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
)

const GAMES = 1000

func main() {
	gamesIds := make(map[string]struct{})

	writeGames(gamesIds)
	writeReviews(gamesIds)
}

func writeGames(gamesIds map[string]struct{}) {
	fullGames, err := os.Open(".data/games.csv")
	if err != nil {
		fmt.Printf("Error opening games file: %v\n", err)
	}
	defer fullGames.Close()

	reduced, err := os.OpenFile(".data/games-reduced.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open reduced games file: %v\n", err)
	}
	defer reduced.Close()

	r := csv.NewReader(fullGames)
	r.FieldsPerRecord = -1
	w := csv.NewWriter(reduced)
	line := 0

	// write header
	record, err := r.Read()
	if err != nil {
		fmt.Printf("Failed to read record: %v\n", err)
	}
	err = w.Write(record)
	if err != nil {
		fmt.Printf("Failed to write record: %v\n", err)
	}
	w.Flush()

	for line < GAMES {
		record, err := r.Read()
		if err != nil {
			fmt.Printf("Failed to read record: %v\n", err)
		}

		err = w.Write(record)
		if err != nil {
			fmt.Printf("Failed to write record: %v\n", err)
		}
		w.Flush()

		line += 1

		if line == 1 {
			continue
		}

		id := record[0]
		gamesIds[id] = struct{}{}
	}
}

func writeReviews(gamesIds map[string]struct{}) {

	fullReviews, err := os.Open(".data/reviews.csv")
	if err != nil {
		fmt.Printf("Error opening reviews file: %v\n", err)
	}
	defer fullReviews.Close()

	reduced, err := os.OpenFile(".data/reviews-reduced.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open reduced reviews file: %v\n", err)
	}
	defer reduced.Close()

	r := csv.NewReader(fullReviews)
	r.FieldsPerRecord = -1
	w := csv.NewWriter(reduced)

	// write header
	record, err := r.Read()
	if err != nil {
		fmt.Printf("Failed to read record: %v\n", err)
	}
	err = w.Write(record)
	if err != nil {
		fmt.Printf("Failed to write record: %v\n", err)
	}
	w.Flush()

	for {
		record, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("Failed to read record: %v\n", err)
		}

		if _, ok := gamesIds[record[0]]; ok {
			err = w.Write(record)
			if err != nil {
				fmt.Printf("Failed to write record: %v\n", err)
			}
			w.Flush()
		}
	}
}
