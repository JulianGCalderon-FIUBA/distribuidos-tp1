package main

import (
	"distribuidos/tp1/server/middleware"
	"distribuidos/tp1/utils"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/rylans/getlang"
)

func main() {
	inputFile, err := os.Open(".data/reviews.csv")
	utils.Expect(err, "failed to open file")
	outputFile, err := os.Create("english_reviews_go.csv")
	utils.Expect(err, "failed to open file")

	reader := csv.NewReader(inputFile)
	reader.FieldsPerRecord = -1
	_, _ = reader.Read()

	writer := csv.NewWriter(outputFile)

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

		info := getlang.FromString(review.Text)
		if info.LanguageName() == "English" {
			err = writer.Write([]string{
				strconv.Itoa(int(review.AppID)),
				review.Text,
			})
			utils.Expect(err, "failed to write file")
		}
	}

	writer.Flush()
}

var emptyReviewTextError = errors.New("review text should not be empty")

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
		err = emptyReviewTextError
		return
	}
	review.Score = middleware.Score(score)

	return
}
