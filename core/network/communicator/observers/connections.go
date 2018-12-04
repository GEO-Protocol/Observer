package observers

import (
	"bufio"
	"errors"
	"geo-observers-blockchain/core/network/external"
	"net"
	"sync"
	"time"
)

var (
	ErrNoObserver = errors.New("no such index")
)

type ConnectionWrapper struct {
	Connection net.Conn
	Writer     *bufio.Writer
	LastUsed   time.Time
}

type ConnectionsMap struct {
	Connections map[*external.Observer]*ConnectionWrapper
	mutex       sync.Mutex
}

func NewConnectionsMap(maxDelay time.Duration) *ConnectionsMap {
	m := &ConnectionsMap{
		Connections: make(map[*external.Observer]*ConnectionWrapper),
	}

	// todo: move into separate code block
	// Auto-cleaner.
	//go func() {
	//	clean := func() {
	//		indexesForRemoving := make([]*external.Observer, 0, len(m.Connections))
	//		for observer, wrapper := range m.Connections {
	//			maxDelayTimeout := time.Now().Add(maxDelay * -1)
	//
	//			if wrapper.LastUsed.Before(maxDelayTimeout) {
	//				indexesForRemoving = append(indexesForRemoving, observer)
	//			}
	//		}
	//
	//		for _, observer := range indexesForRemoving {
	//			delete(m.Connections, observer)
	//		}
	//	}
	//
	//	for {
	//		time.Sleep(maxDelay)
	//
	//		m.mutex.Lock()
	//		clean()
	//		m.mutex.Unlock()
	//	}
	//}()

	return m
}

func (cm *ConnectionsMap) Get(observer *external.Observer) (*ConnectionWrapper, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	w, isPresent := cm.Connections[observer]
	if !isPresent {
		return nil, ErrNoObserver
	}

	w.LastUsed = time.Now()
	return w, nil
}

func (cm *ConnectionsMap) Set(observer *external.Observer, conn net.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.Connections[observer] = &ConnectionWrapper{
		Connection: conn,
		Writer:     bufio.NewWriter(conn),
		LastUsed:   time.Now(),
	}
}

func (cm *ConnectionsMap) Delete(observer *external.Observer, conn *net.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.Connections, observer)
}
