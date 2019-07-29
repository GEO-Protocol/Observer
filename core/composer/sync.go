package composer

import (
	"fmt"
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/events"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os/exec"
	"time"
)

// processSyncRequest begins chain synchronisation.
//
// WARN:
// It is expected, that current chain is validated and no orphan blocks are present.
func (c *Composer) processSyncRequest(request *syncRequest) (err errors.E) {

	// Emit request about synchronisation completeness on function execution end,
	// regardless which result was received.
	defer func() {
		c.OutgoingEvents.SyncFinished <- &events.SyncFinished{Error: err}
	}()

	c.log().Info("Chain synchronisation started")

	// In some cases, when observer wasn't finished correctly, it is possible,
	// that directory for temporary data contains potentially incomplete snapshots from previous session.
	// This data should not be taken nto account in this session,
	// because it might not correspond to current network state.
	if err = c.dropChainBackupIfAny(); err != nil {
		return
	}

	// Last block should be kept for external chain validation purposes.
	// In case if current chain has no any data - it is expected, that genesis block is present,
	// so at least one block is present and no error should be dropped.
	currentChainLastBlock, err := request.Chain.LastBlock()
	if err != nil {
		return
	}

	if err = c.requestActualChainStateFromOtherObservers(currentChainLastBlock); err != nil {
		return
	}

	observersClusters, observersResponses, err := c.performObserversResponsesClustering()
	if err != nil {
		// It is possible, that no one observer will respond.
		// This case could be normal network behaviour in case
		// if current observer is the first attendee of the network.
		if err.Error() == errors.NoResponseReceived {
			err = nil // Mark current error as normal behaviour.
			c.log().WithFields(log.Fields{"ResponsesCount": 0}).Info("Synchronisation is done")
			c.log().Warn("Independent chain started!")
		}

		// No sync is possible.
		// Current chain data should be verified and in case of success -- considered as valid.
		// todo: add current chain validation.

		request.LastProcessedEventChan <- request.Tick
		return
	}

	if err = c.processSyncDependingOnRemoteObserversResponses(
		request.Chain, currentChainLastBlock, observersClusters, observersResponses, request); err != nil {
		return

	} else {
		c.log().WithFields(
			log.Fields{"ResponsesCount": len(observersResponses)},
		).Info("Synchronisation is done")
		return
	}
}

func (c *Composer) processSyncDependingOnRemoteObserversResponses(
	chain *chain.Chain, lastChainBlock *block.Signed, observersClusters []*observersCluster,
	observersResponses map[uint16]*responses.ChainInfo, request *syncRequest) (e errors.E) {

	if len(observersClusters) == 0 {
		e = errors.AppendStackTrace(errors.NoResponseReceived)

		request.LastProcessedEventChan <- request.Tick
		return
	}

	majorityLastBlockHash := observersClusters[0].LastBlockHash
	if lastChainBlock.Body.Hash.Compare(majorityLastBlockHash) {
		if settings.Conf.Debug {
			c.log().WithFields(log.Fields{
				"LastBlockHash":         lastChainBlock.Body.Hash.Hex(),
				"MajorityLastBlockHash": majorityLastBlockHash.Hex(),
				"Cluster":               observersClusters[0].ObserversIndexes,
			}).Debug("Reported highest block hash is equal to current highest block hash. No sync is needed")
		}

		request.LastProcessedEventChan <- request.Tick
		return
	}

	// Both sync method (see further) applies their changes only to temporary data.
	chainBackupFilename, e := c.createOrReplaceChainBackup()
	if e != nil {
		return
	}

	// Checking how many observers are included in biggest cluster received.
	// In case if this number is greater than expected consensus number -
	// whole chain rewrite is considered as allowed.
	biggestCluster := observersClusters[0]
	strongConsensusIsAchieved := len(biggestCluster.ObserversIndexes) >= settings.ObserversConsensusCount

	if strongConsensusIsAchieved {
		return c.processConsensusProvenSync(chainBackupFilename, chain, biggestCluster, request)

	} else {
		return c.processIncrementalSync(chainBackupFilename, chain, observersClusters, request)
	}
}

// processConsensusProvenSync runs synchronisation with
func (c *Composer) processConsensusProvenSync(
	chainBackupFilename string, currentChain *chain.Chain,
	majority *observersCluster, request *syncRequest) (e errors.E) {

	// It is possible, that observer, that would be selected for chain sync would not respond.
	// In this case algorithm should try to connect with another observer from pool of majority.
	// This process should be repeat until chain would not be synchronised,
	// or all observers would be queried.

	// In the same time, it is very important to pick observers in truly random manner.
	// Otherwise, there is a possibility to compromise hashes collection process,
	// get profitable position in observers list and then try to spoof the data.
	//
	// This attack can not forge the final chain data,
	// because it would be detected and mitigated by the further checks,
	// but it can greatly increase traffic used and slow down the whole process.

	remainedObserversIndexes := majority.ObserversIndexes[:]

	// Shuffling observers indexes.
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(remainedObserversIndexes), func(i, j int) {
		remainedObserversIndexes[i], remainedObserversIndexes[j] =
			remainedObserversIndexes[j], remainedObserversIndexes[i]
	})

	// todo: comment needed
	doAttempt := func(attempt, totalAttemptsCount int) (e errors.E, nextAttemptAllowed bool) {
		// Popping the last index.
		var nextObserverIndex uint16
		nextObserverIndex, remainedObserversIndexes =
			remainedObserversIndexes[len(remainedObserversIndexes)-1],
			remainedObserversIndexes[:len(remainedObserversIndexes)-1]

		c.log().Info(fmt.Sprint(
			"Picked observer ", nextObserverIndex, ". Attempt ", attempt, " of ", totalAttemptsCount, "."))

		params, e := c.rsyncParams(nextObserverIndex, chainBackupFilename,
			true, request.Tick.ObserversConfiguration)
		if e != nil {
			// In case of error - it should be considered as fatal error,
			// and no other observer selection should be performed.
			nextAttemptAllowed = false
			return
		}

		// todo: replace rsync with custom protocol.
		// 		 rsync is known to have buffer overflow vulnerabilities,
		//		 so it must be replaced with custom proven data exchange protocol,
		//		 or jailed into safe sandbox.
		cmd := exec.Command("rsync", params...)
		err := cmd.Run()
		if err != nil {
			e = errors.AppendStackTrace(err)
			nextAttemptAllowed = true
			return
		}

		e = c.validateFullyRewrittenChain(chainBackupFilename)
		if e != nil {
			// Current retrieved chain data is invalid.
			// Algorithm must try to sync with another observer.
			nextAttemptAllowed = true
			return
		}

		var lastTimeFrameWindow *ticker.EventTimeFrameStarted
		select {
		case lastTimeFrameWindow = <-request.TicksChannel:
			{
			}
		case <-time.After(settings.AverageBlockGenerationTimeRange * 4):
			e = errors.New("Ticker doesn't generates tick events. " +
				"Can't determine when to load last block.")
			nextAttemptAllowed = false
			return
		}

		// Load last generated block.
		cmd = exec.Command("rsync", params...)
		err = cmd.Run()
		if err != nil {
			e = errors.AppendStackTrace(err)
			nextAttemptAllowed = true
			return
		}

		e = c.validateLastBlock(chainBackupFilename, currentChain)
		if e != nil {
			// Current retrieved chain data is invalid.
			// Algorithm must try to sync with another observer.
			nextAttemptAllowed = true
			return
		}

		request.LastProcessedEventChan <- lastTimeFrameWindow
		return
	}

	attempt := 1
	totalAttemptsCount := len(majority.ObserversIndexes)

	for {
		if len(remainedObserversIndexes) == 0 {
			e = errors.New("No more observers are available. Sync failed.")
			return
		}

		e, nextAttemptAllowed := doAttempt(attempt, totalAttemptsCount)
		if e != nil {
			if nextAttemptAllowed {
				// In case of error - it should be reported,
				// and next observer should be queried.
				c.log().Error(e.Error())

				attempt += 1
				continue
			}

			// Next attempt is not allowed (fatal error occurred).
			// Previous chain version must be restored from the backup.
			e = c.restoreChainBackup(chainBackupFilename)
			if e != nil {
				return e
			}

			return c.dropChainBackupIfAny()
		}

		c.log().Info("Chain sync has been finished.")

		// Temporary copy was synchronised with the rest of network and validated.
		// Now it should replace original data.
		return c.restoreChainBackup(chainBackupFilename)
	}
}

func (c *Composer) processIncrementalSync(
	chainBackupFilename string, currentChain *chain.Chain,
	observersClusters []*observersCluster, request *syncRequest) (err errors.E) {

	// todo: comment needed
	attemptToSync := func(attemptNumber, totalAttemptsCount int, observerIndex uint16) (e errors.E, nextAttemptAllowed bool) {
		description := log.Fields{
			"ObserverIndex": observerIndex,
			"Attempt":       fmt.Sprint(attemptNumber, "/", totalAttemptsCount),
		}
		c.log().WithFields(description).Info("Attempt to sync with remote observer")

		params, e := c.rsyncParams(observerIndex, chainBackupFilename,
			true, request.Tick.ObserversConfiguration)
		if e != nil {
			// In case of error - it should be considered as fatal error,
			// and no other observer selection should be performed.
			nextAttemptAllowed = false
			return
		}

		// todo: replace rsync with custom protocol.
		// 		 rsync is known to have buffer overflow vulnerabilities,
		//		 so it must be replaced with custom proven data exchange protocol,
		//		 or jailed into safe sandbox.
		cmd := exec.Command("rsync", params...)
		err := cmd.Run()
		if err != nil {
			e = errors.AppendStackTrace(err)
			nextAttemptAllowed = true
			return
		}

		e = c.validateIncrementallySynchronisedChain(chainBackupFilename, currentChain)
		if e != nil {
			// Current retrieved chain data is invalid.
			// Algorithm must try to sync with another observer.
			nextAttemptAllowed = true
			return
		}

		// Wait for last block to be generated.
		var lastTimeFrameWindow *ticker.EventTimeFrameStarted
		select {
		case lastTimeFrameWindow = <-request.TicksChannel:
			{
			}
		case <-time.After(settings.AverageBlockGenerationTimeRange * 4):
			e = errors.New("Ticker doesn't generates tick events. " +
				"Can't determine when to load last block.")
			nextAttemptAllowed = false
			return
		}

		// Load last generated block.
		cmd = exec.Command("rsync", params...)
		err = cmd.Run()
		if err != nil {
			e = errors.AppendStackTrace(err)
			nextAttemptAllowed = true
			return
		}

		e = c.validateLastBlock(chainBackupFilename, currentChain)
		if e != nil {
			// Current retrieved chain data is invalid.
			// Algorithm must try to sync with another observer.
			nextAttemptAllowed = true
			return
		}

		request.LastProcessedEventChan <- lastTimeFrameWindow
		return
	}

	// todo: comment needed
	processCluster := func(cluster *observersCluster) (e errors.E, nextAttemptAllowed bool) {

		// It is possible, that observer, that would be selected for chain sync would not respond.
		// In this case algorithm should try to connect with another observer from pool of majority.
		// This process should be repeat until chain would not be synchronised,
		// or all observers would be queried.

		// In the same time, it is very important to pick observers in truly random manner.
		// Otherwise, there is a possibility to compromise hashes collection process,
		// get profitable position in observers list and then try to spoof the data.
		//
		// This attack can not forge the final chain data,
		// because it would be detected and mitigated by the further checks,
		// but it can greatly increase traffic used and slow down the whole process.
		remainedObserversIndexes := cluster.ObserversIndexes[:]

		// Shuffling observers indexes.
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(remainedObserversIndexes), func(i, j int) {
			remainedObserversIndexes[i], remainedObserversIndexes[j] =
				remainedObserversIndexes[j], remainedObserversIndexes[i]
		})

		attempt := 1
		totalAttemptsCount := len(remainedObserversIndexes)

		for {
			if len(remainedObserversIndexes) == 0 {
				e = errors.New("No more observers are available. Sync failed.")
				return
			}

			// Popping the last index.
			var nextObserverIndex uint16
			nextObserverIndex, remainedObserversIndexes =
				remainedObserversIndexes[len(remainedObserversIndexes)-1],
				remainedObserversIndexes[:len(remainedObserversIndexes)-1]

			e, nextAttemptAllowed = attemptToSync(attempt, totalAttemptsCount, nextObserverIndex)
			if e != nil {
				if !nextAttemptAllowed {
					// Sync has encountered fatal error.
					return

				} else {
					c.logError(e, "Failed to sync with observer")

					attempt += 1
					continue
				}

			} else {
				// Sync was done successfully.
				return
			}
		}
	}

	// todo: comment needed
	processIncrementalSync := func() {
		c.log().Info("No strong consensus has been achieved. Incremental sync started.")

		for _, cluster := range observersClusters {
			e, nextAttemptAllowed := processCluster(cluster)
			if e != nil {
				description := describeClusterOnError(cluster, err)
				c.log().WithFields(description).Error("Cluster processing failed.")

				if !nextAttemptAllowed {
					c.log().Error("No next attempt could be processed. Exit.")
					return

				} else {
					continue
				}

			} else {
				c.log().Info("Incremental sync finished.")
				return
			}
		}
	}

	processIncrementalSync()
	if err != nil {
		return
	}

	// Temporary copy was synchronised with the rest of network and validated.
	// Now it should replace original data.
	return c.restoreChainBackup(chainBackupFilename)
}

// requestActualChainStateFromOtherObservers enqueues chain info request for sending to each observer in the network.
func (c *Composer) requestActualChainStateFromOtherObservers(currentLastBlock *block.Signed) (err errors.E) {
	select {
	case c.OutgoingRequests.ChainInfo <- requests.NewChainInfoRequest(currentLastBlock.Body.Index):
	default:
		err = errors.AppendStackTrace(errors.ChannelTransferringFailed)
		return
	}

	// Sleep enough time to collect responses from other observers about their version of chain top.
	time.Sleep(settings.ComposerSynchronisationTimeRange)
	return
}

func (c *Composer) rsyncParams(
	pickedObserverIndex uint16, destinationFilename string, chainRewriteIsAllowed bool,
	currentObserversConfiguration *external.Configuration) (params []string, e errors.E) {

	observer := currentObserversConfiguration.Observers[pickedObserverIndex]

	module := "chain"
	// todo: drop it on beta release
	if settings.Conf.Debug {
		module = module + fmt.Sprint(pickedObserverIndex)
	}

	params = make([]string, 0, 4)
	params = append(params,
		"-crdt",
		fmt.Sprint("rsync://", observer.Host, ":873/", module, "/chain.dat"),
		"./"+destinationFilename)

	if !chainRewriteIsAllowed {
		params = append(params, "--append")
	}

	return
}

func describeCluster(cluster *observersCluster) (fields log.Fields) {
	if cluster == nil {
		return log.Fields{}
	}

	fields = log.Fields{
		"Cluster.LastBlockHash":    cluster.LastBlockHash.Hex(),
		"Cluster.ObserversIndexes": cluster.ObserversIndexes,
	}
	return
}

func describeClusterOnError(cluster *observersCluster, e errors.E) (fields log.Fields) {
	fields = describeCluster(cluster)

	if e != nil {
		fields["err"] = e.Error()
		fields["stacktrace"] = e.StackTrace()
	}

	return
}
