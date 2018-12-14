package timer

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

type Timer struct {
	OutgoingEventsTimeFrameEnd chan *EventTimeFrameEnd
	OutgoingRequestsTimeFrames chan *requests.RequestTimeFrames
	IncomingRequestsTimeFrames chan *requests.RequestTimeFrames
	OutgoingResponsesTimeFrame chan *responses.ResponseTimeFrame
	IncomingResponsesTimeFrame chan *responses.ResponseTimeFrame

	internalEventsBus chan interface{}

	confReporter *external.Reporter

	// Time left for the next time frame.
	// By default, it is equal to the block generation time duration,
	// but might be set to lover value after Sync() call.
	nextFrameTimestamp time.Time
	isTickerRunning    bool

	frame *EventTimeFrameEnd
}

func New(reporter *external.Reporter) *Timer {
	initialConfiguration, _ := reporter.GetCurrentConfiguration()

	return &Timer{
		// Outgoing events channel is not buffered.
		// It is better to lost timer tick, than process several ticks
		// one by one without any delay, that might be considered as, malicious behaviour.
		OutgoingEventsTimeFrameEnd: make(chan *EventTimeFrameEnd),

		OutgoingRequestsTimeFrames: make(chan *requests.RequestTimeFrames, 1),
		OutgoingResponsesTimeFrame: make(chan *responses.ResponseTimeFrame, 1),

		// On synchronization stage,
		// timer should be able to collect up to MAX OBSERVERS count of responses.
		IncomingResponsesTimeFrame: make(chan *responses.ResponseTimeFrame, common.ObserversMaxCount),
		IncomingRequestsTimeFrames: make(chan *requests.RequestTimeFrames, 1),

		// Internal events bus is used to control and to interrupt internal events loop.
		internalEventsBus: make(chan interface{}),

		confReporter: reporter,

		frame: &EventTimeFrameEnd{
			Index:         math.MaxUint16,
			Configuration: initialConfiguration,
		},
	}
}

func (t *Timer) Run(errors chan error) {
	shortLoop := func() {
		//t.log().Debug("Short loop started")

		select {

		// todo: reconfigure frames on external observers configuration change

		case timeFramesRequest := <-t.IncomingRequestsTimeFrames:
			{
				err := t.processTimeFrameRequest(timeFramesRequest)
				common.SendErrorIfAny(err, errors)
			}

		case event := <-t.internalEventsBus:
			{
				err := t.processInternalEvent(event)
				common.SendErrorIfAny(err, errors)
			}
		}
	}

	fullLoop := func() {
		//t.log().Debug("Long loop started")

		select {

		// todo: reconfigure frames on external observers configuration change

		case _ = <-time.After(t.nextFrameTimeLeft()):
			{
				t.processTick()
			}

		case timeFramesRequest := <-t.IncomingRequestsTimeFrames:
			{
				err := t.processTimeFrameRequest(timeFramesRequest)
				common.SendErrorIfAny(err, errors)
			}

		case event := <-t.internalEventsBus:
			{
				err := t.processInternalEvent(event)
				common.SendErrorIfAny(err, errors)
			}
		}
	}

	// Attempt to sync with other observers before any operations processing.
	// It is asynchronous operation so it must be launched in goroutine to not block internal events loop
	// and make it possible to respond to requests.
	go t.syncWithOtherObservers()

	for {
		if t.isTickerRunning {
			fullLoop()

		} else {
			shortLoop()
		}
	}
}

func (t *Timer) syncWithOtherObservers() {

	setNextTick := func(timeLeft time.Duration) {
		t.nextFrameTimestamp = time.Now().Add(timeLeft)
		t.internalEventsBus <- &EventTickerStarted{}
	}

	// Request external observers for their current time frames data.
	// Timer would process all collected responses and
	// would adjust it's own configuration in accordance to the majority.
	t.OutgoingRequestsTimeFrames <- &requests.RequestTimeFrames{}

	// Wait for all responses to be present
	// todo: prevent blocking if all responses are present already.
	//  (change response waiting logic)
	time.Sleep(time.Second * 5)

	majorityOfTTLs, err := t.findMajorityOfFrameResponses()
	if err != nil {
		t.log().Debug("Synchronisation is done, but NO RESPONSES was took into account")
		setNextTick(common.AverageBlockGenerationTimeRange)

	} else {
		t.log().Debug("Synchronisation is done, ", len(majorityOfTTLs), " responds is took into account")
		averageTTL := t.processMajorityAndCalculateAverageNextFrameTTL(majorityOfTTLs)
		setNextTick(time.Nanosecond * time.Duration(averageTTL))
	}
}

func (t *Timer) processInternalEvent(event interface{}) error {
	switch event.(type) {
	case *EventTickerStarted:
		{
			t.isTickerRunning = true
			return nil
		}

	default:
		return common.ErrNilParameter
	}
}

func (t *Timer) processTimeFrameRequest(request *requests.RequestTimeFrames) error {
	var response *responses.ResponseTimeFrame

	if t.isTickerRunning {
		response = responses.NewResponseTimeFrame(
			request,
			t.frame.Index,
			uint64(t.nextFrameTimeLeft().Nanoseconds()))

	} else {
		response = responses.NewResponseTimeFrame(
			request,
			t.frame.Index,
			0)
	}

	select {
	case t.OutgoingResponsesTimeFrame <- response:
		return nil

	default:
		return common.ErrChannelTransferringFailed
	}
}

func (t *Timer) processTick() {
	nextFrameNumber := t.frame.Index + 1
	if nextFrameNumber == common.ObserversMaxCount {
		nextFrameNumber = 0
	}

	// Important!
	// New event always must replace previous one.
	// Do not update event's fields directly.
	t.frame = &EventTimeFrameEnd{
		Index:         nextFrameNumber,
		Configuration: t.frame.Configuration,
	}

	select {
	case t.OutgoingEventsTimeFrameEnd <- t.frame:
		{
		}
	default:
		return
	}
}

func (t *Timer) nextFrameTimeLeft() (d time.Duration) {
	//defer func() {
	//	t.log().Debug(d)
	//}()

	timeLeft := t.nextFrameTimestamp.Sub(time.Now())
	if timeLeft <= 0 {
		t.nextFrameTimestamp = time.Now().Add(
			common.AverageBlockGenerationTimeRange).Add(
			(timeLeft * time.Nanosecond) * -1)

		return t.nextFrameTimeLeft()
	}

	return timeLeft
	//
	//if t.nextFrameNanosecondsLeft.Nanoseconds() < common.AverageBlockGenerationTimeRange.Nanoseconds() {
	//	tmp := t.nextFrameNanosecondsLeft
	//	t.nextFrameNanosecondsLeft = common.AverageBlockGenerationTimeRange
	//	return tmp
	//}
	//
	//return common.AverageBlockGenerationTimeRange
}

func (t *Timer) reconfigureFrames(e *external.EventConfigurationChanged) {
	// todo: implement
}

func (t *Timer) processMajorityAndCalculateAverageNextFrameTTL(TTLs []uint64) uint64 {
	if len(TTLs) == 0 {
		return 0
	}

	// todo: Sort the sequence.
	// todo: Accept only majority of votes, median
	//  (K%, K == consensus count).

	var total uint64 = 0
	for _, ttl := range TTLs {
		total += ttl
	}

	return total / uint64(len(TTLs))
}

func (t *Timer) findMajorityOfFrameResponses() (majority []uint64, err error) {
	totalResponsesPresent := len(t.IncomingResponsesTimeFrame)
	if totalResponsesPresent == 0 {
		return nil, common.ErrEmptySequence
	}

	rates := make(map[uint16]*[]uint64)

	var (
		topFrameIndex      uint16
		topFrameVotesCount = 0
		currentTTLsCount   = 0
	)

	now := time.Now()
	for i := 0; i < totalResponsesPresent; i++ {
		vote := <-t.IncomingResponsesTimeFrame
		frameIndex := vote.FrameIndex

		TTLs, isPresent := rates[frameIndex]

		// todo: add comment
		timeOffset := now.Sub(vote.Received).Nanoseconds()

		var correctedNanosecondsLeft int64 = 0
		correctedNanosecondsLeft = int64(common.AverageBlockGenerationTimeRange) +
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

	nextFrameIndex := topFrameIndex + 1
	if nextFrameIndex >= common.ObserversMaxCount {
		nextFrameIndex = 0
	}

	t.frame = &EventTimeFrameEnd{Index: nextFrameIndex}
	return *m, nil
}

func (t *Timer) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "Timer"})
}
