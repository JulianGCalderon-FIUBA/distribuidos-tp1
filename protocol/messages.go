package protocol

import "fmt"

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

type Results interface {
	ToCSV() []string
}

// Results Messages
type Q1Results struct {
	Windows int
	Linux   int
	Mac     int
}

func (q1 Q1Results) ToCSV() []string {
	q := "Query 1"
	w := fmt.Sprintf("windows: %v", q1.Windows)
	l := fmt.Sprintf("linux: %v", q1.Linux)
	m := fmt.Sprintf("mac: %v", q1.Mac)

	return []string{q, w, l, m}
}

type Q2Results struct {
	TopN []string
}

func (q2 Q2Results) ToCSV() []string {
	s := make([]string, len(q2.TopN)+1)
	q := "Query 2"
	s = append(s, q)
	s = append(s, q2.TopN...)
	return s
}

type Q3Results struct {
	TopN []string
}

func (q3 Q3Results) ToCSV() []string {
	s := make([]string, len(q3.TopN)+1)
	q := "Query 3"
	s = append(s, q)
	s = append(s, q3.TopN...)
	return s
}

type Q4Results struct {
	Name string
	EOF  bool
}

func (q4 Q4Results) ToCSV() []string {
	s := make([]string, 2)
	q := "Query 4"
	s = append(s, q)
	s = append(s, q4.Name)
	return s
}

type Q5Results struct {
	Percentile90 []string
}

func (q5 Q5Results) ToCSV() []string {
	s := make([]string, len(q5.Percentile90)+1)
	q := "Query 5"
	s = append(s, q)
	s = append(s, q5.Percentile90...)
	return s
}
