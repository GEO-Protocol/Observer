package tests

type ObserverAddress struct {
	Host    string
	GEOPort uint16
}

var (
	Observers = []ObserverAddress{
		{Host: "127.0.0.1", GEOPort: 4000},
		{Host: "127.0.0.1", GEOPort: 4001},
		{Host: "127.0.0.1", GEOPort: 4002},
		{Host: "127.0.0.1", GEOPort: 4003},
	}
)
