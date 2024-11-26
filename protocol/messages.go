package protocol

import (
	"distribuidos/tp1/middleware"
	"strconv"
)

// Sent by the client to initiate a request
type RequestHello struct {
	GameSize   uint64
	ReviewSize uint64
}

// Sent by the connection handler to accept a client's request
type AcceptRequest struct {
	ClientID uint64
}

// Data Handler Messages

// Sent by the client to present itself to the data handler
type DataHello struct {
	ClientID uint64
}

// Sent by the data handler to accept a client
type DataAccept struct{}

// Sent by the client to the data handler
type Batch struct {
	Data []byte
}

// Sent by the client to indicate that it has finished sending data
type Finish struct{}

// Results Messages

type Result interface {
	Header() []string
	ToCSV() [][]string
	Number() int
}

type Q1Result struct {
	Windows int
	Linux   int
	Mac     int
}

type Q2Result struct {
	TopN []middleware.GameStat
}

type Q3Result struct {
	TopN []middleware.GameStat
}

type Q4Result struct {
	Games []middleware.GameStat
}

type Q5Result struct {
	Percentile90 []middleware.GameStat
}

type Q4Finish struct {
}

func (q Q1Result) Header() []string { return []string{"Linux", "Mac", "Windows"} }
func (q Q2Result) Header() []string { return []string{"AppID", "Name", "Average playtime forever"} }
func (q Q3Result) Header() []string { return []string{"AppID", "Name", "Reviews"} }
func (q Q4Result) Header() []string { return []string{"AppID", "Name", "Reviews"} }
func (q Q5Result) Header() []string { return []string{"AppID", "Name", "Reviews"} }

func (q Q4Finish) Header() []string { return []string{} }

func (q Q1Result) Number() int { return 1 }
func (q Q2Result) Number() int { return 2 }
func (q Q3Result) Number() int { return 3 }
func (q Q4Result) Number() int { return 4 }
func (q Q5Result) Number() int { return 5 }
func (q Q4Finish) Number() int { return 4 }

func (q Q1Result) ToCSV() [][]string {
	return [][]string{{
		strconv.Itoa(q.Linux),
		strconv.Itoa(q.Mac),
		strconv.Itoa(q.Windows),
	}}
}

func (q Q2Result) ToCSV() [][]string { return GameStatsToCSV(q.TopN) }
func (q Q3Result) ToCSV() [][]string { return GameStatsToCSV(q.TopN) }
func (q Q4Result) ToCSV() [][]string { return GameStatsToCSV(q.Games) }
func (q Q5Result) ToCSV() [][]string { return GameStatsToCSV(q.Percentile90) }

func (q Q4Finish) ToCSV() [][]string { return [][]string{} }

func GameStatsToCSV(s []middleware.GameStat) [][]string {
	res := make([][]string, 0)
	for _, s := range s {
		res = append(res, []string{
			strconv.Itoa(int(s.AppID)),
			s.Name,
			strconv.Itoa(int(s.Stat)),
		})
	}
	return res
}

func AllResultTypes() []Result {
	return []Result{
		Q1Result{},
		Q2Result{},
		Q3Result{},
		Q4Result{},
		Q5Result{},
	}
}
