package protocol

import (
	"distribuidos/tp1/middleware"
	"fmt"
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

type Results interface {
	ToStringArray() []string
}

// Results Messages
type Q1Results struct {
	Windows int
	Linux   int
	Mac     int
}

func (q1 Q1Results) ToStringArray() []string {
	w := fmt.Sprintf("windows: %v\n", q1.Windows)
	l := fmt.Sprintf("linux: %v\n", q1.Linux)
	m := fmt.Sprintf("mac: %v\n", q1.Mac)

	return []string{w, l, m}
}

type Q2Results struct {
	TopN []middleware.GameStat
}

func (q2 Q2Results) ToStringArray() []string {
	res := make([]string, 0)
	for _, s := range q2.TopN {
		res = append(res, fmt.Sprintf("%v,%v,%v\n", s.AppID, s.Name, s.Stat))
	}
	return res
}

type Q3Results struct {
	TopN []middleware.GameStat
}

func (q3 Q3Results) ToStringArray() []string {
	res := make([]string, 0)
	for _, s := range q3.TopN {
		res = append(res, fmt.Sprintf("%v,%v,%v\n", s.AppID, s.Name, s.Stat))
	}
	return res
}

type Q4Results struct {
	AppID uint64
	Name  string
	Count int
	EOF   bool
}

func (q4 Q4Results) ToStringArray() []string {
	res := make([]string, 0)
	res = append(res, fmt.Sprintf("%v,%v,%v\n", q4.AppID, q4.Name, q4.Count))
	return res
}

type Q5Results struct {
	Percentile90 []middleware.GameStat
}

func (q5 Q5Results) ToStringArray() []string {
	res := make([]string, 0)
	for _, s := range q5.Percentile90 {
		res = append(res, fmt.Sprintf("%v,%v,%v\n", s.AppID, s.Name, s.Stat))
	}
	return res
}
