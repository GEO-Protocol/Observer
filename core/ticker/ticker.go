package ticker

import (
	errors2 "geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

// todo: think about time synchronisation once in a week/month,
//       to prevent permanent delta increasing and time frames shifting.

const (
	// WARN!
	// Initial time frame index can't be 0, because it is valid index.
	// It also can't be < 0, because of uint16.
	// It is expected, that observers count would never exceed uint16.
	// So it seems, that ideal initial value for kInitialTimeFrameIndex should be MAX uint16.
	// On the next increment it would be set to 0.
	kInitialTimeFrameIndex = math.MaxUint16
)

// ToDo: synchronisation mechanics needs huge testing.
//       It works well enough for beta and internal GEO Client development, and even for the test-net,
//       but IT IS NOT READY for the production usage.

type Ticker struct {
	OutgoingEventsTimeFrameEnd         chan *EventTimeFrameStarted
	OutgoingRequestsTimeFrames         chan *requests.SynchronisationTimeFrames
	IncomingRequestsTimeFrames         chan *requests.SynchronisationTimeFrames
	OutgoingResponsesTimeFrame         chan *responses.TimeFrame
	IncomingResponsesTimeFrame         chan *responses.TimeFrame
	IncomingRequestsTimeFrameCollision chan *requests.TimeFrameCollision

	// Internal events bus is used for controlling internal events loop.
	// For example, in case if synchronisation with external observers is finished,
	// and ticker might be started.
	internalEventsBus chan interface{}

	// External observers configuration reporter.
	confReporter *external.Reporter

	// Time left for the next time frame.
	// By default, it is equal to the block generation time duration,
	// but might be set to lover value after Sync() call.
	nextFrameTimestamp time.Time

	// Time when synchronisation must be finished.
	synchronisationDeadlineTimestamp time.Time

	// If true - then ticker is synchronized and is generating new ticks.
	// By default is set to "false", because it is expected,
	// that ticker would be synchronized first.
	isTickerRunning bool

	// Frame == current time window.
	// Total frames count == total observers count in current observers configuration.
	// Each one frame is related to the corresponding observer.
	// Frames are used as logical time windows for observers to emit blocks.
	// For example, for observer 0 it's related time frame has number 0.
	// Current frame index monotonically increases over time.
	// In case of current frame index reaches last observer -
	// process begins from the beginning.
	frame *EventTimeFrameStarted
	//
	//// Invalid frames reports
	//ObserversReportedInvalidIndex map[uint16]bool
}

func New(reporter *external.Reporter) *Ticker {
	initialConfiguration, _ := reporter.GetCurrentConfiguration()

	return &Ticker{
		// Outgoing events channel is not buffered.
		// It is better to lost ticker tick, than process several ticks
		// one by one without any delay, that might be considered as, malicious behaviour.
		OutgoingEventsTimeFrameEnd: make(chan *EventTimeFrameStarted),
		OutgoingRequestsTimeFrames: make(chan *requests.SynchronisationTimeFrames, 1),
		OutgoingResponsesTimeFrame: make(chan *responses.TimeFrame, 1),

		// On synchronization stage,
		// ticker should be able to collect up to MAX OBSERVERS count of responses.
		IncomingResponsesTimeFrame:         make(chan *responses.TimeFrame, settings.ObserversMaxCount),
		IncomingRequestsTimeFrames:         make(chan *requests.SynchronisationTimeFrames, 1),
		IncomingRequestsTimeFrameCollision: make(chan *requests.TimeFrameCollision, 1),

		// Internal events bus is used to control and to interrupt internal events loop.
		internalEventsBus: make(chan interface{}, 1),

		confReporter: reporter,

		frame: &EventTimeFrameStarted{
			Index:                  kInitialTimeFrameIndex,
			ObserversConfiguration: initialConfiguration,
		},

		//ObserversReportedInvalidIndex: make(map[uint16]bool),
	}
}

func (t *Ticker) Run(errors chan error) {
	shortLoop := func() {
		select {

		// todo: reconfigure frames on external observers configuration change

		//case timeFramesRequest := <-t.IncomingRequestsTimeFrames:
		//	errors2.SendErrorIfAny(
		//		t.processTimeFrameRequest(timeFramesRequest), errors)

		case event := <-t.internalEventsBus:
			err := t.processInternalEvent(event)
			errors2.SendErrorIfAny(err, errors)
		}
	}

	fullLoop := func() {
		select {

		// todo: reconfigure frames on external observers configuration change

		case _ = <-time.After(t.nextFrameTimeLeft()):
			e := t.processTick()
			if e != nil {
				errors2.SendErrorIfAny(e.Error(), errors) // todo: replace by internal errors processing (errors.E)
			}

		case timeFramesRequest := <-t.IncomingRequestsTimeFrames:
			err := t.processTimeFrameRequest(timeFramesRequest)
			errors2.SendErrorIfAny(err, errors)

		//case timeFramesCollisionRequest := <-t.IncomingRequestsTimeFrameCollision:
		//	errors2.SendErrorIfAny(
		//		t.processTimeFrameCollisionRequest(timeFramesCollisionRequest), errors)

		case event := <-t.internalEventsBus:
			err := t.processInternalEvent(event)
			errors2.SendErrorIfAny(err, errors)
		}
	}

	// WARN!
	// Whole synchronisation flow MUST perform faster than one block generation timeout.
	// At the moment, current logic does not support time frames synchronisation
	// when 2 or more blocks was generated during synchronisation.
	//
	// todo: add support of short blocks timeouts.

	var (
		kMinimalTimeFramesExchangeTimeoutSeconds = 2
		kMinimalAppropriateTimeoutSeconds        = int(settings.AverageBlockGenerationTimeRange.Seconds()) -
			kMinimalTimeFramesExchangeTimeoutSeconds
	)
	if int(settings.TickerSynchronisationTimeRange.Seconds()) >= kMinimalAppropriateTimeoutSeconds {
		// todo: replace panic
		panic(ErrInvalidSynchronisationTimeout)
	}

	// Attempt to sync with other observers before any operations processing.
	// It is asynchronous operation, so it must be launched in goroutine
	// to not block internal events loop and make it possible to respond to requests from core.
	// Static assert check.
	go t.syncWithOtherObservers()

	for {
		if t.isTickerRunning {
			fullLoop()

		} else {
			shortLoop()
		}
	}
}

func (t *Ticker) Restart() {
	t.internalEventsBus <- &eventSyncRequested{}
}

func (t *Ticker) InitialTimeFrame() (frame *EventTimeFrameStarted, e errors2.E) {
	return t.generateFrame(0)
}

func (t *Ticker) syncWithOtherObservers() {
	setNextTick := func(offset time.Duration) {
		t.nextFrameTimestamp = time.Now().Add(offset)

		// Interrupt internal loop, so this change would be processed.
		t.internalEventsBus <- &eventTickerStarted{}
	}

	t.log().Info("Synchronization started")

	nextFrameOffset, nextFrameIndex, responsesCollected, err := t.processSync()
	if err == errors2.EmptySequence {
		t.log().WithFields(
			log.Fields{"ResponsesCount": 0}).Info("Synchronisation is done")
		t.log().Warn("Independent time frames flow started")

		// Use default block generation time range.
		t.frame = &EventTimeFrameStarted{Index: nextFrameIndex}
		setNextTick(settings.AverageBlockGenerationTimeRange)

	} else {
		t.log().WithFields(
			log.Fields{"ResponsesCount": responsesCollected}).Info("Synchronisation is done")

		t.frame = &EventTimeFrameStarted{Index: nextFrameIndex}
		setNextTick(time.Nanosecond * time.Duration(nextFrameOffset))
	}
}

func (t *Ticker) processSync() (
	timeOffsetNanoseconds uint64, nextFrameIndex uint16, collectedResponsesCount uint16, err error) {

	// Request external observers for their current time frames data.
	// Ticker would process all collected responses and
	// would adjust it's own configuration in accordance to the majority.
	select {
	case t.OutgoingRequestsTimeFrames <- &requests.SynchronisationTimeFrames{}:
	default:
		err = errors2.ChannelTransferringFailed
		return
	}

	t.synchronisationDeadlineTimestamp = time.Now().Add(settings.TickerSynchronisationTimeRange)

	for {
		if time.Now().After(t.synchronisationDeadlineTimestamp) {
			break
		}

		time.Sleep(time.Millisecond * 50)
		if len(t.IncomingResponsesTimeFrame) == settings.ObserversMaxCount {
			// There is no reason to wait longer.
			// All responses has been collected.
			break
		}
	}

	return t.processMajorityOfFrameResponses()
}

func (t *Ticker) processInternalEvent(event interface{}) error {
	switch event.(type) {
	case *eventTickerStarted:
		t.isTickerRunning = true
		return nil

	case *eventSyncRequested:
		t.isTickerRunning = false
		t.syncWithOtherObservers()
		return nil

	default:
		return errors2.NilParameter
	}
}

// processTimeFrameRequest schedules sending of response
// with information about CURRENT time frame index and amount of nanoseconds to it's change.
// In case if ticker is in sync mode - it also adds amount of nanoseconds to the sync. finish.
func (t *Ticker) processTimeFrameRequest(request *requests.SynchronisationTimeFrames) error {
	if !t.isTickerRunning {
		if t.synchronisationDeadlineTimestamp.Second() == 0 {
			// In case if ticker is stopped, but is not in synchronisation phase -
			// no time frame response should be returned, but it's also not an error.
			return nil
		}
	}

	conf, err := t.confReporter.GetCurrentConfiguration()
	if err != nil {
		return err
	}

	var (
		response          *responses.TimeFrame
		nextFrameTimeLeft = t.nextFrameTimeLeft()
	)

	if nextFrameTimeLeft.Seconds() < 3 {
		nextFrameTimeLeft += settings.AverageBlockGenerationTimeRange
	}

	if t.frame.Index == kInitialTimeFrameIndex {
		response = responses.NewTimeFrame(
			request,
			conf.CurrentObserverIndex,
			0,
			uint64(t.nextFrameTimeLeft().Nanoseconds()))

	} else {
		response = responses.NewTimeFrame(
			request,
			conf.CurrentObserverIndex,
			t.frame.Index,
			uint64(t.nextFrameTimeLeft().Nanoseconds()))
	}

	select {
	case t.OutgoingResponsesTimeFrame <- response:
		return nil

	default:
		return errors2.ChannelTransferringFailed
	}
}

//func (t *Ticker) processTimeFrameCollisionRequest(request *requests.TimeFrameCollision) (err error) {
//	if !t.isTickerRunning {
//		return
//	}
//
//	// todo: collision report might be used for draining the node.
//	//       add some filter map[observer] -> reports count per time.
//
//	t.ObserversReportedInvalidIndex[request.ObserverIndex()] = true
//	if len(t.ObserversReportedInvalidIndex) > settings.ObserversConsensusCount {
//		t.log().Debug("!!! Collision detected")
//		t.syncWithOtherObservers()
//	}
//
//	return
//}

func (t *Ticker) processTick() (e errors2.E) {
	nextFrameIndex := t.frame.Index + 1
	if nextFrameIndex == uint16(settings.ObserversMaxCount) {
		nextFrameIndex = 0
	}

	// WARN!
	// Event instance must always be replaced!
	// Do not update event's fields directly.
	t.frame, e = t.generateFrame(nextFrameIndex)
	if e != nil {
		return
	}

	select {
	case t.OutgoingEventsTimeFrameEnd <- t.frame:
	default:
		t.log().Error("tick transfer error")
	}

	return
}

// nextFrameTimeLeft returns time duration to the next time frame.
// Might be called several times during frame processing:
// each time the result would be les than the previous,
// so it is ok for events to interrupt internal events loop.
func (t *Ticker) nextFrameTimeLeft() (d time.Duration) {
	timeLeft := t.nextFrameTimestamp.Sub(time.Now())
	if timeLeft <= 0 {
		t.nextFrameTimestamp = time.Now().Add(
			settings.AverageBlockGenerationTimeRange).Add(
			timeLeft * time.Nanosecond * -1)

		return t.nextFrameTimeLeft()
	}

	return timeLeft
}

func (t *Ticker) reconfigureFrames(e *external.EventConfigurationChanged) {
	// todo: implement on the ethereum connection implementation stage
}

// processMajorityAndCalculateAverageNextFrameTTL returns average time offset.
// Returns 0 in case if no offset is present in majorityOfTimeOffsets.
func (t *Ticker) processMajorityAndCalculateAverageNextFrameTTL(majorityOfTimeOffsets []uint64) uint64 {
	if len(majorityOfTimeOffsets) == 0 {
		return 0
	}

	// todo: Accept only majority of votes, (median, consensus)
	//       (K%, K == consensus count).

	var total uint64 = 0
	for _, ttl := range majorityOfTimeOffsets {
		total += ttl
	}

	return total / uint64(len(majorityOfTimeOffsets))
}

// processMajorityOfFrameResponses processes collected time frames responses,
// finds the majority of the responses, checks if majority has reached the consensus,
// and collects time offsets of the observers, that has fit into the majority.
// Returns error in case if consensus has not been reached.
func (t *Ticker) processMajorityOfFrameResponses() (
	timeOffsetNanoseconds uint64, nextFrameIndex uint16, collectedResponsesCount uint16, err error) {
	collectedResponsesCount = uint16(len(t.IncomingResponsesTimeFrame))
	if collectedResponsesCount == 0 {
		return 0, 0, 0, errors2.EmptySequence
	}

	rates := make(map[uint16]*[]uint64)

	var (
		i                  uint16
		topFrameIndex      uint16
		topFrameVotesCount = 0
		currentTTLsCount   = 0
		now                = time.Now()
	)

	for i = 0; i < collectedResponsesCount; i++ {
		vote := <-t.IncomingResponsesTimeFrame
		frameIndex := vote.FrameIndex

		TTLs, isPresent := rates[frameIndex]

		// todo: add comment
		timeOffset := now.Sub(vote.Received).Nanoseconds()

		var correctedNanosecondsLeft int64 = 0
		correctedNanosecondsLeft = int64(settings.AverageBlockGenerationTimeRange) +
			int64(vote.NanosecondsLeft) -
			int64(timeOffset)

		if isPresent {
			*TTLs = append(*TTLs, uint64(correctedNanosecondsLeft))
			currentTTLsCount = len(*TTLs)

		} else {
			rates[frameIndex] = &[]uint64{uint64(correctedNanosecondsLeft)}
			currentTTLsCount = 1
		}

		if currentTTLsCount > topFrameVotesCount {
			topFrameIndex = frameIndex
			topFrameVotesCount = currentTTLsCount
		}
	}

	m, _ := rates[topFrameIndex]
	timeOffsetNanoseconds = t.processMajorityAndCalculateAverageNextFrameTTL(*m)

	frameOffset := 0
	if timeOffsetNanoseconds > uint64(settings.AverageBlockGenerationTimeRange.Nanoseconds()) {
		frameOffset = 1
	}

	nextFrameIndex = topFrameIndex + uint16(frameOffset)
	if nextFrameIndex >= uint16(settings.ObserversMaxCount) {
		nextFrameIndex = 0
	}

	return
}

func (t *Ticker) generateFrame(index uint16) (frame *EventTimeFrameStarted, e errors2.E) {
	observersConfiguration, err := t.confReporter.GetCurrentConfiguration()
	if err != nil {
		e = errors2.New(err.Error())
		return
	}

	validationStageEndTimestamp :=
		time.Now().Add(settings.AverageBlockGenerationTimeRange).Add(
			-settings.BlockValidationStageFinalizingTimeRange)

	blockGenerationStageEndTimestamp :=
		validationStageEndTimestamp.Add(-(settings.BlockGenerationStageFinalizingTimeRange))

	frame = &EventTimeFrameStarted{
		Index:                            index,
		ObserversConfiguration:           observersConfiguration,
		ValidationStageEndTimestamp:      validationStageEndTimestamp,
		BlockGenerationStageEndTimestamp: blockGenerationStageEndTimestamp,
	}
	return
}
func (t *Ticker) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Ticker"})
}
