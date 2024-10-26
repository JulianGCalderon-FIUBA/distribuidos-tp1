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

	lingua "github.com/pemistahl/lingua-go"
)

func isEnglish(text string) bool {
	languages := []lingua.Language{
		lingua.English,
		lingua.Spanish,
	}

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(languages...).
		Build()

	lang, _ := detector.DetectLanguageOf(text)

	return lang == lingua.English
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Not enough arguments")
	}

	inputArg := os.Args[1]
	outputArg := os.Args[2]

	inputFile, err := os.Open(inputArg)
	utils.Expect(err, "failed to open file")
	outputFile, err := os.Create(outputArg)
	utils.Expect(err, "failed to open file")

	reader := csv.NewReader(inputFile)
	reader.FieldsPerRecord = -1
	writer := csv.NewWriter(outputFile)

	header, err := reader.Read()
	utils.Expect(err, "failed to read file")
	err = writer.Write(header)
	utils.Expect(err, "failed to write file")

	for {
		record, err := reader.Read()
		if errors.Is(err, &csv.ParseError{}) {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		utils.Expect(err, "failed to read file")

		review, err := reviewFromFullRecord(record)
		if err != nil {
			continue
		}

		if isEnglish(review.Text) {
			err = writer.Write(record)
			utils.Expect(err, "failed to write file")
		}
	}

	writer.Flush()
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
