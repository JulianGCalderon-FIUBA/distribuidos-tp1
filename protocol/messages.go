package protocol

type MessageTag uint8

const (
	RequestHelloTag MessageTag = iota
	AcceptRequestTag
	DataHelloTag
	DataAcceptTag
	PrepareTag
	BatchTag
	FinishTag
)

// Connection Handler Messages

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

// Sent by the client to indicate that it will send a file
type Prepare struct{}

// Sent by the client to the data handler
type Batch struct {
	Lines [][]byte
}

// Sent by the client to indicate that it has finished sending data
type Finish struct{}

type Game struct {
	AppID                  int
	Name                   string
	ReleaseDate            string
	Windows                bool
	Mac                    bool
	Linux                  bool
	AveragePlaytimeForever int
	Genres                 string
	// EstimatedOwners         string
	// PeakCCU                 int
	// RequiredAge             int
	// Price                   float64
	// Unknown                 int
	// DiscountDLCCount        int
	// AboutTheGame            string
	// SupportedLanguages      string
	// FullAudioLanguages      string
	// Reviews                 string
	// HeaderImage             string
	// Website                 string
	// SupportUrl              string
	// SupportEmail            string
	// MetacriticScore         int
	// MetacriticUrl           string
	// UserScore               int
	// Positive                int
	// Negative                int
	// ScoreRank               float64
	// Achievements            int
	// Recommendations         int
	// Notes                   string
	// AveragePlaytimeTwoWeeks int
	// MedianPlaytimeForever   int
	// MedianPlaytimeTwoWeeks  int
	// Developers              string
	// Publishers              string
	// Categories              string
	// Tags                    string
	// Screenshots             string
	// Movies                  string
}
type Review struct {
	AppID   int
	AppName string
	Text    string
	Score   int
}
