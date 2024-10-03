package middleware_test

import (
	"distribuidos/tp1/server/middleware"
	"reflect"
	"testing"
)

func TestSerialization(t *testing.T) {
	messages := []any{
		middleware.Game{
			AppID:                  5,
			AveragePlaytimeForever: 3,
			Windows:                true,
			Mac:                    false,
			Linux:                  false,
			ReleaseYear:            2009,
			Name:                   "la saturacion del pipeline",
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
	}

	for _, msg := range messages {
		buf, _ := middleware.Serialize(msg)

		precv := reflect.New(reflect.TypeOf(msg))
		_ = middleware.DeserializeInto(buf, precv.Interface())
		recv := precv.Elem().Interface()

		if !reflect.DeepEqual(msg, recv) {
			t.Fatalf("expected %#+v, but received %#+v", msg, recv)
		}
	}
}
