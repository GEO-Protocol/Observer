package chain

import (
	"bytes"
	"fmt"
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	kTempFilesExtension = ".temp"
)

type Composer struct {
	OutgoingEventsSynchronizationRequested chan *EventSynchronizationRequested
	OutgoingEventsSynchronizationFinished  chan *EventSynchronizationFinished
	IncomingEventsTimeFrameChanged         chan *ticker.EventTimeFrameEnd // todo: not used
	OutgoingRequestsChainTop               chan *requests.ChainTop
	IncomingResponsesChainTop              chan *responses.ChainTop

	// todo: add channels for collisions notifications

	internalEventsBus chan interface{}

	observers *external.Reporter
	settings  *settings.Settings
}

func NewComposer(reporter *external.Reporter, settings *settings.Settings) (composer *Composer) {
	composer = &Composer{
		OutgoingEventsSynchronizationRequested: make(chan *EventSynchronizationRequested, 1),
		OutgoingEventsSynchronizationFinished:  make(chan *EventSynchronizationFinished, 1),
		IncomingEventsTimeFrameChanged:         make(chan *ticker.EventTimeFrameEnd, 1),
		OutgoingRequestsChainTop:               make(chan *requests.ChainTop, 1),
		IncomingResponsesChainTop:              make(chan *responses.ChainTop, 1024),

		internalEventsBus: make(chan interface{}, 1),

		observers: reporter,
		settings:  settings,
	}
	return
}

func (c *Composer) Run(globalErrorsFlow chan<- error) {

	processErrorIfAny := func(err errors.E) {
		if err != nil {
			// todo: add support of errors.E
			globalErrorsFlow <- err.Error()
		}
	}

	for {
		select {
		case event := <-c.internalEventsBus:
			processErrorIfAny(
				c.processInternalEvent(event))
		}
	}
}

// SyncChain initialises chain synchronisation with majority of other observers.
// In case if majority of the observers would agree on some version of the chain (reach consensus) -
// current chain would be synchronised with the majority even in case if current chain version DIFFERS
// in some (or even majority of blocks) blocks (full node synchronisation).
//
// In case if consensus can't be reached - incremental synchronisation would be performed:
// node would only try to append corresponding data from the majority of present observers,
// even if them are not represents consensus majority. This is needed to make it possible
// to bootstrap the chain.
//
// In other cases - node would not synchronize automatically to prevent sensitive data loss.
// In all such cases node must wait for consensus presence or sync with some other observer node
// on it's own risk.
//
// Returns channel from which synchronisation result should be read.
//
// WARN: This method must be called only from the core:
// synchronisation should be performed only when block producer is stopped!
// Only node's core has enough context to stop blocks producer and launch synchronisation.
func (c *Composer) SyncChain(chain *Chain) chan *EventSynchronizationFinished {
	// Internal events loop is used to perform all
	// storage-related operations in goroutine safe manner.
	c.internalEventsBus <- &EventSynchronizationRequested{Chain: chain}
	return c.OutgoingEventsSynchronizationFinished
}

func (c *Composer) processInternalEvent(event interface{}) (err errors.E) {
	switch event.(type) {
	case *EventSynchronizationRequested:
		return c.processNodeSynchronisation(event.(*EventSynchronizationRequested).Chain)

	default:
		err = errors.AppendStackTrace(errors.UnexpectedEvent)
	}

	return
}

func (c *Composer) processNodeSynchronisation(chain *Chain) (err errors.E) {
	defer func() {
		c.OutgoingEventsSynchronizationFinished <- &EventSynchronizationFinished{Error: err}
	}()

	processError := func(error errors.E) {
		c.logError(error, "Can't synchronize chain data with other observers")
	}

	// todo: restore chain from backup if there is backup file, but no chain file.

	err = c.dropTemporaryDataIfAny()
	if err != nil {
		processError(err)
		return
	}

	lastBlock, err := chain.LastBlock()
	if err != nil {
		processError(err)
		return
	}

	err = c.requestActualChainState(lastBlock)
	if err != nil {
		processError(err)
		return
	}

	consensusObserversIndexes, proposedTopBlockHash, proposedCurrentBlockHash, err := c.observersWithSameLastBlockHash()
	if err != nil {
		processError(err)
		return
	}

	err = c.processSync(chain, consensusObserversIndexes, proposedTopBlockHash, proposedCurrentBlockHash, lastBlock)
	if err != nil {
		processError(err)
		return
	}

	if c.settings.Debug {
		c.log().Debug("Synchronisation is done successfully.")
	}
	return
}

func (c *Composer) requestActualChainState(topBlock *block.Signed) (err errors.E) {
	// WARN: No current block hash must be included in to the request
	//       Otherwise malicious observer might fake the response with the hash that is expected,
	//       by simply substituting it it to the response.

	select {
	case c.OutgoingRequestsChainTop <- requests.NewChainTop(topBlock.Body.Index):
	default:
		err = errors.AppendStackTrace(errors.ChannelTransferringFailed)
		return
	}

	time.Sleep(common.ComposerSynchronisationTimeRange)
	return
}

// todo: ensure responses from remote observers are signed.
//       Otherwise spoofing is possible.
func (c *Composer) observersWithSameLastBlockHash() (consensusObserversIndexes []uint16,
	proposedTopBlockHash, proposedCurrentBlockHash *hash.SHA256Container, e errors.E) {

	totalResponsesReceived := len(c.IncomingResponsesChainTop)
	if totalResponsesReceived == 0 {
		c.log().Warn("No one response was received from observers")
		c.log().Warn("Synchronisation cancelled, independent chain started!")
		return nil, nil, nil, nil
	}

	topBlockHashClusters := make(map[hash.SHA256Container]*[]uint16)
	requestedBlockHashClusters := make(map[uint16]*hash.SHA256Container)

	for i := 0; i < totalResponsesReceived; i++ {
		response := <-c.IncomingResponsesChainTop

		observersIndexes, isPresent := topBlockHashClusters[*response.TopBlockHash]
		if isPresent {
			*observersIndexes = append(*observersIndexes, response.ObserverIndex())

		} else {
			topBlockHashClusters[*response.TopBlockHash] = &[]uint16{response.ObserverIndex()}
		}

		requestedBlockHashClusters[response.ObserverIndex()] = response.RequestedBlockHash
	}

	var (
		majority          = 0
		majorityObservers = make([]uint16, 0, 0)
		topHash           *hash.SHA256Container
	)

	// WARN: Consensus should not be taken into account here.
	//       Otherwise, initial consensus would never consolidate.
	for key, observersIndexes := range topBlockHashClusters {
		totalObserversInCluster := len(*observersIndexes)
		if totalObserversInCluster > majority {
			majority = totalObserversInCluster
			majorityObservers = *observersIndexes
			topHash = &key
		}
	}

	requestedBlockHash := requestedBlockHashClusters[majorityObservers[0]]
	for _, observerIndex := range majorityObservers {
		h, isPresent := requestedBlockHashClusters[observerIndex]
		if !isPresent || bytes.Compare(h.Bytes[:], requestedBlockHash.Bytes[:]) != 0 {
			e = errors.AppendStackTrace(errors.NoConsensus)
			return
		}
	}

	return majorityObservers, requestedBlockHash, topHash, nil
}

func (c *Composer) processSync(actualChain *Chain, observersIndexes []uint16,
	proposedTopBlockHash, proposedHashForCurrentBlock *hash.SHA256Container, lastBlock *block.Signed) (e errors.E) {

	if len(observersIndexes) == 0 {
		return
	}

	if bytes.Compare(lastBlock.Body.Hash.Bytes[:], proposedTopBlockHash.Bytes[:]) == 0 {
		// Reported highest block hash is equal to current highest block hash.
		// No need to launch synchronisation.
		return
	}

	// todo: process case when majority says current highest block is orphan
	//       if rewrite s allowed - do a full sync,
	//       otherwise - report error and give detailed instructions what to do.

	allowChainRewrite := len(observersIndexes) >= common.ObserversConsensusCount
	if !allowChainRewrite {
		// todo: lock segment of file

		if bytes.Compare(lastBlock.Body.Hash.Bytes[:], proposedHashForCurrentBlock.Bytes[:]) != 0 {
			// Current highest block seems to be an orphan.
			// NO data rewrite should be done, otherwise, cluster of malicious observers might cooperate
			// and try to spoof original valid data on some valid observers

			c.log().WithField(
				"Description",
				"The consensus on the reported current chain state was not reached! "+
					"The major observers cluster is "+fmt.Sprint(len(observersIndexes))+
					" items long: "+fmt.Sprint(observersIndexes)+

					"Original data would NOT be rewritten!"+
					"You must retry to synchronise with the majority after some period of time, "+
					"when consensus of observers would be available back.",
			).Warn("No information about current chain state was received from other observers.")

			e = errors.AppendStackTrace(errors.NoConsensus)
			return
		}
	}

	return c.syncChainAndValidate(actualChain, observersIndexes, allowChainRewrite)
}

func (c *Composer) syncChainAndValidate(actualChain *Chain,
	observersIndexes []uint16, allowChainRewrite bool) (e errors.E) {

	temporaryCopyFilename, e := c.backupChainData()
	if e != nil {
		return
	}

	params, e := c.rsyncParams(observersIndexes, temporaryCopyFilename, allowChainRewrite)
	if e != nil {
		return
	}

	cmd := exec.Command("rsync", params...)
	err := cmd.Run()
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	e = c.checkRetrievedData(actualChain, temporaryCopyFilename, allowChainRewrite)
	if err != nil {
		return
	}

	c.rewriteChainDataFromTmpCopy(temporaryCopyFilename)
	if err != nil {
		return
	}

	return c.dropTemporaryDataIfAny()
}

func (c *Composer) rsyncParams(observersIndexes []uint16, tmpCopyFilename string,
	allowChainRewrite bool) (params []string, e errors.E) {

	conf, err := c.observers.GetCurrentConfiguration()
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	// It is very important to select final observer, from which the chain wold be downloaded,
	// in truly random manner. Otherwise, there is a possibility to compromise hashes collection process,
	// get profitable position in observers list and then try to spoof the data.
	//
	// This attack can not forge the final chain data,
	// because it would be detected and mitigated by the further checks,
	// but it can greatly increase traffic used and slow down whole the process.
	rand.Seed(time.Now().Unix())

	index := rand.Intn(len(observersIndexes))
	observer := conf.Observers[observersIndexes[index]]

	module := "chain"
	// todo: drop it on beta release
	if c.settings.Debug {
		module = module + fmt.Sprint(observersIndexes[index])
	}

	params = make([]string, 0, 4)
	params = append(params,
		"-crdt",
		fmt.Sprint("rsync://", observer.Host, ":873/", module, "/chain.dat"),
		"./"+tmpCopyFilename)

	if !allowChainRewrite {
		params = append(params, "--append")
	}
	return
}

func (c *Composer) checkRetrievedData(actualChain *Chain, tmpCopyFilename string, wholeChainValidation bool) (e errors.E) {
	chainCandidate, err := NewChain(tmpCopyFilename)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	if wholeChainValidation {
		return c.checkBlocksSequence(chainCandidate, 1)

	} else {
		return c.checkBlocksSequence(chainCandidate, actualChain.Height())

	}
}

func (c *Composer) checkBlocksSequence(candidate *Chain, offset uint64) (e errors.E) {
	var previousBlockHash hash.SHA256Container
	if offset == 0 {
		e = errors.AppendStackTrace(errors.InvalidParameter)
		return

	} else if offset == 1 {
		previousBlockHash = candidate.GenerateGenesisBlock().Body.Hash

	} else {
		previousBlock, err := candidate.BlockAt(offset - 1)
		if err != nil {
			e = errors.AppendStackTrace(errors.ValidationFailed)
			return
		}

		previousBlockHash = previousBlock.Body.Hash
	}

	totalBlocksCount := candidate.Height()
	for ; offset < totalBlocksCount; offset++ {
		nextBlock, err := candidate.BlockAt(offset)
		if err != nil {
			e = errors.AppendStackTrace(errors.ValidationFailed)
			return
		}

		reportedHash := nextBlock.Body.Hash

		err = nextBlock.Body.SortInternalSequences()
		if err != nil {
			e = errors.AppendStackTrace(errors.ValidationFailed)
			return
		}

		err = nextBlock.Body.UpdateHash(previousBlockHash)
		if err != nil {
			e = errors.AppendStackTrace(errors.ValidationFailed)
			return
		}

		if nextBlock.Body.Hash.Compare(&reportedHash) == false {
			e = errors.AppendStackTrace(errors.ValidationFailed)
		}

		previousBlockHash = nextBlock.Body.Hash
	}

	return
}

func (c *Composer) backupChainData() (tmpCopyFilename string, e errors.E) {
	tmpCopyFilename = fmt.Sprint(DataFilePath, "_", time.Now().Unix(), kTempFilesExtension)

	cmd := exec.Command("cp", "--reflink", DataFilePath, tmpCopyFilename)
	err := cmd.Run()

	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return
}

func (c *Composer) rewriteChainDataFromTmpCopy(tmpCopyFilename string) (e errors.E) {
	cmd := exec.Command("rm", DataFilePath)
	err := cmd.Run()
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	cmd = exec.Command("mv", tmpCopyFilename, DataFilePath)
	err = cmd.Run()
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return
}

func (c *Composer) dropTemporaryDataIfAny() (e errors.E) {
	files, err := filepath.Glob(DataPath + "*" + kTempFilesExtension)
	if err != nil {
		e = errors.AppendStackTrace(err)
		c.logError(e, "Can't find all temporary files.")
		return
	}

	for _, f := range files {
		if err := os.Remove(f); err != nil {
			e = errors.AppendStackTrace(err)
			c.logError(e, "Can't remove temporary file: "+f)
			return
		}
	}

	return
}

func (c *Composer) logError(err errors.E, message string) {
	if c.settings.Debug {
		c.log().WithField("StackTrace", err.StackTrace()).Error(message, ", ", err.Error())
	} else {
		c.log().Error(message, ", ", err.Error())
	}
}

func (c *Composer) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Composer"})
}
