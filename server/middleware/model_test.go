package middleware_test

import (
	"distribuidos/tp1/server/middleware"
	"reflect"
	"testing"
)

func TestSerialization(t *testing.T) {
	messages := []middleware.Message{
		&middleware.Game{
			AppID:                  5,
			AveragePlaytimeForever: 3,
			Windows:                true,
			Mac:                    false,
			Linux:                  false,
			ReleaseDate: middleware.Date{
				Day:   3,
				Month: 1,
				Year:  2009,
			},
			Name: "la saturacion del pipeline",
			Genres: []string{
				"distribuidos",
				"roca",
			},
		},
		&middleware.Review{
			AppID: 5,
			Score: 7,
			Text:  "muy bueno!!",
		},
		&middleware.BatchGame{[]middleware.Game{{
			AppID:                  5,
			AveragePlaytimeForever: 3,
			Windows:                true,
			Mac:                    false,
			Linux:                  false,
			ReleaseDate: middleware.Date{
				Day:   3,
				Month: 1,
				Year:  2009,
			},
			Name: "la saturacion del pipeline",
			Genres: []string{
				"distribuidos",
				"roca",
			}},
		},
		},
		&middleware.BatchReview{[]middleware.Review{
			{AppID: 5,
				Score: 7,
				Text:  "muy bueno!!"},
		}},
	}

	for _, entity := range messages {
		buf, err := entity.Serialize()
		if err != nil {
			t.Fatalf("failed to serialize: %v", err)
		}

		var deserialized middleware.Message
		switch entity.(type) {
		case *middleware.Game:
			deserialized = &middleware.Game{}
		case *middleware.Review:
			deserialized = &middleware.Review{}
		case *middleware.BatchGame:
			deserialized = &middleware.BatchGame{}
		case *middleware.BatchReview:
			deserialized = &middleware.BatchReview{}
		}

		_, err = deserialized.Deserialize(buf)
		if err != nil {
			t.Fatalf("failed to deserialize %v", err)
		}

		if !reflect.DeepEqual(entity, deserialized) {
			t.Fatalf("expected %#+v, but received %#+v", entity, deserialized)
		}
	}
}
