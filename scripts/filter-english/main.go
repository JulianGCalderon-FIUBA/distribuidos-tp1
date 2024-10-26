package main

import (
	"distribuidos/tp1/middleware"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"

	lingua "github.com/pemistahl/lingua-go"
)

const MAX_THREADS = 8

func run(input, output chan []string) {
	log.Println("Starting language filter")

	languages := []lingua.Language{
		lingua.English,
		lingua.Spanish,
	}
	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()

	for record := range input {
		review, err := reviewFromFullRecord(record)
		if err != nil {
			continue
		}
		lang, _ := detector.DetectLanguageOf(review.Text)
		if lang == lingua.English {
			output <- record
		}
	}

	log.Println("Finished language filter")
}

func read(filePath string, output chan []string) {
	log.Println("Starting reader")

	file, err := os.Open(filePath)
	utils.Expect(err, "failed to open file")
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	for {
		record, err := reader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		utils.Expect(err, "failed to read file")

		output <- record
	}

	log.Println("Finished reader")

	close(output)
}

func write(filePath string, input chan []string) {
	log.Println("Starting writer")

	file, err := os.Create(filePath)
	utils.Expect(err, "failed to open file")
	writer := csv.NewWriter(file)

	for record := range input {
		err = writer.Write(record)
		utils.Expect(err, "failed to write file")
	}

	writer.Flush()
	utils.Expect(writer.Error(), "failed to flush file")

	log.Println("Finished writer")
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Not enough arguments")
	}

	inputArg := os.Args[1]
	outputArg := os.Args[2]

	input := make(chan []string)
	output := make(chan []string)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		read(inputArg, input)
		wg.Done()
	}()

	wg.Add(MAX_THREADS)
	for range MAX_THREADS {
		go func() {
			run(input, output)
			wg.Done()
		}()
	}

	// We wait until all filter threads have finished
	// before closing the channel, ensuring no data is lost
	go func() {
		wg.Wait()
		close(output)
	}()

	write(outputArg, output)
}

func reviewFromFullRecord(record []string) (review middleware.Review, err error) {
	if len(record) < 4 {
		err = fmt.Errorf("expected 4 fields, got %v", len(record))
		return
	}
	appId, err := strconv.Atoi(record[0])
	if err != nil {
		return
	}
	score, err := strconv.Atoi(record[3])
	if err != nil {
		return
	}

	review.AppID = uint64(appId)
	review.Text = record[2]
	if review.Text == "" {
		err = errors.New("review text should not be empty")
		return
	}
	review.Score = middleware.Score(score)

	return
}
