package settings

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	l "github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
)

var (
	Conf *Settings
)

const (
	KObserversMaxCount = 1024
)

var (
	ObserversMaxCount       = KObserversMaxCount
	ObserversConsensusCount = 758

	// This period of time is delegated to the observer for block generation and distribution.
	// It is expected, that block generation would be almost timeless,
	// and the rest of time would be spent for the block distribution.
	AverageBlockGenerationTimeRange = time.Minute

	// Time range during which remote observers might respond with their ticker states.
	// WARN: This value must be at least 3 times less, than block generation time range.
	// todo: ensure check of this constraint on ticker starting.
	TickerSynchronisationTimeRange = time.Second * 20

	// Time range during which remote observers might respond on synchronisation requests.
	// WARN: This value must be at least 3 times less, than block generation time range.
	// todo: ensure check of this constraint on ticker starting.
	ComposerSynchronisationTimeRange = time.Second * 20

	// This period of time is used as a buffer time window:
	// during this time window observer does not accepts any external events or messages,
	// and prepares to process next ticker tick.
	//
	// This time window MUST be shorter than block generation time range,
	// because it would be subtracted from the AverageBlockGenerationTimeRange.
	// todo: drop this parameter at all.
	BlockGenerationSilencePeriod = time.Second * 10

	// todo: sync with the GEO engine
	GEOTransactionMaxParticipantsCount = 700
)

var (
	OutputNetworkObserversSenderDebug      = false
	OutputNetworkObserversSenderWarnings   = false
	OutputNetworkObserversReceiverDebug    = false
	OutputNetworkObserversReceiverWarnings = false
	OutputBlocksProducerDebug              = false
)

type networkInterface struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type gns struct {
	networkInterface

	// todo: replace this parameter by GSN Middleware checker
	PKeyFilePath string `json:"pkey"`
}

type observers struct {
	Network networkInterface `json:"network"`
	GNS     gns              `json:"gns"`
}

type nodes struct {
	Network networkInterface `json:"network"`
}

type Settings struct {
	Debug     bool      `json:"debug"`
	Observers observers `json:"observers"`
	Nodes     nodes     `json:"nodes"`
}

func LoadSettings() error {
	Conf = &Settings{}

	bytes, err := ioutil.ReadFile("conf.json")
	if err != nil {
		return errors.New("Can't read configuration. Details: " + err.Error())
	}

	err = json.Unmarshal(bytes, Conf)
	if err != nil {
		return errors.New("Can't read configuration. Details: " + err.Error())
	}

	parseFlags()
	return nil
}

func parseFlags() {
	mode := flag.String(
		"mode", "normal",
		"Specifies execution mode of the observer. \n"+
			"Possible options are: \n"+
			"\n"+
			"`debug-cluster` - observer would be configured to work in group of 5 observers in total."+
			"This mode is for debug/tests purposes.\n"+
			"\n"+
			"`normal` - observer would be launched as usual and work in production-ready configuration.")
	flag.Parse()

	if *mode == "debug-cluster" {
		fmt.Print("MODE DebugCluster is ON.\n\n")

		ObserversMaxCount = 4
		ObserversConsensusCount = 3
		AverageBlockGenerationTimeRange = time.Second * 10
		TickerSynchronisationTimeRange = time.Second * 2
		ComposerSynchronisationTimeRange = time.Second * 2
		BlockGenerationSilencePeriod = time.Second * 2

		OutputNetworkObserversSenderDebug = false
		OutputNetworkObserversSenderWarnings = false
		OutputNetworkObserversReceiverDebug = false
		OutputNetworkObserversReceiverWarnings = false
		OutputBlocksProducerDebug = true
	}
}

func log() *l.Entry {
	return l.WithFields(l.Fields{"prefix": "Conf"})
}
