package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

const GAMES = 1000

func main() {
	gamesIds := make([]string, 0)

	gamesIds = writeGames(gamesIds)
	writeReviews(gamesIds)
}

func writeGames(gamesIds []string) []string {
	fullGames, err := os.Open(".data/games.csv")
	if err != nil {
		fmt.Printf("Error opening games file: %v", err)
	}
	defer fullGames.Close()

	reduced, err := os.OpenFile(".data/games-reduced.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open reduced games file: %v", err)
	}
	defer reduced.Close()

	r := csv.NewReader(fullGames)
	r.FieldsPerRecord = 40
	w := csv.NewWriter(reduced)
	line := 0

	// write header
	record, err := r.Read()
	if err != nil {
		fmt.Printf("Failed to read record: %v", err)
	}
	err = w.Write(record)
	if err != nil {
		fmt.Printf("Failed to write record: %v", err)
	}
	w.Flush()

	for line < GAMES {
		record, err := r.Read()
		if err != nil {
			fmt.Printf("Failed to read record: %v", err)
		}

		err = w.Write(record)
		if err != nil {
			fmt.Printf("Failed to write record: %v", err)
		}
		w.Flush()

		line += 1

		if line == 1 {
			continue
		}

		id := record[0]
		gamesIds = append(gamesIds, id)
	}

	return gamesIds
}

func writeReviews(gamesIds []string) {

	fullReviews, err := os.Open(".data/reviews.csv")
	if err != nil {
		fmt.Printf("Error opening reviews file: %v", err)
	}
	defer fullReviews.Close()

	reduced, err := os.OpenFile(".data/reviews-reduced.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open reduced games file: %v", err)
	}
	defer reduced.Close()

	r := csv.NewReader(fullReviews)
	r.FieldsPerRecord = 5
	w := csv.NewWriter(reduced)

	// write header
	record, err := r.Read()
	if err != nil {
		fmt.Printf("Failed to read record: %v", err)
	}
	err = w.Write(record)
	if err != nil {
		fmt.Printf("Failed to write record: %v", err)
	}
	w.Flush()

	for {
		record, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("Failed to read record: %v", err)
		}

		if slices.Contains(gamesIds, record[0]) {
			err = w.Write(record)
			if err != nil {
				fmt.Printf("Failed to write record: %v", err)
			}
			w.Flush()
		}
	}
}
