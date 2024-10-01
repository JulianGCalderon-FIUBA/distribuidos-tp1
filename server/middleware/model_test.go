package middleware_test

import (
	"distribuidos/tp1/server/middleware"
	"reflect"
	"testing"
)

func TestSerialization(t *testing.T) {
	messages := []middleware.Game{
		{
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
	}

	for _, game := range messages {
		buf, err := game.Serialize()
		if err != nil {
			t.Fatalf("failed to serialize: %v", err)
		}

		deserialized_game := middleware.Game{}
		err = deserialized_game.Deserialize(buf)
		if err != nil {
			t.Fatalf("failed to deserialize %v", err)
		}

		if !reflect.DeepEqual(game, deserialized_game) {
			t.Fatalf("expected %#+v, but received %#+v", game, deserialized_game)
		}
	}
}
