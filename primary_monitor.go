package litefs

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrNotReady = errors.New("awaiting first event")
	ErrClosed   = errors.New("PrimaryMonitor closed")
)

// PrimaryMonitor monitors the current primary status of the LiteFS cluster.
type PrimaryMonitor struct {
	es    *EventSubscription
	ready chan struct{}
	m     sync.RWMutex

	isPrimary bool
	hostname  string
	err       error
}

// NewPrimaryMonitor returns a new *PrimaryMonitor.
func NewPrimaryMonitor() *PrimaryMonitor {
	pm := &PrimaryMonitor{
		es:    SubscribeEvents(),
		ready: make(chan struct{}),
	}

	go pm.run()

	return pm
}

// WaitReady blocks until ctx expires or a response or error has been received
// from the local LiteFS node. IsPrimary and Hostname will return errors until
// this method returns nil.
func (pm *PrimaryMonitor) WaitReady(ctx context.Context) error {
	select {
	case <-pm.ready:
		pm.m.RLock()
		defer pm.m.RUnlock()
		return pm.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsPrimary reports whether the local node is primary node in the cluster. An
// error is returned if this method is called before data has been received
// from the local LiteFS node (see WaitReady). If an error is encountered while
// communicating with the local node, that error, along with the most recent
// IsPrimary value will be returned.
func (pm *PrimaryMonitor) IsPrimary() (bool, error) {
	select {
	case <-pm.ready:
	default:
		return false, ErrNotReady
	}

	pm.m.RLock()
	defer pm.m.RUnlock()

	return pm.isPrimary, pm.err
}

// Hostname reports the name of the current primary node in the cluster. An
// error is returned if this method is called before data has been received
// from the local LiteFS node (see WaitReady). If an error is encountered while
// communicating with the local node, that error, along with the most recent
// Hostname value will be returned.
func (pm *PrimaryMonitor) Hostname() (string, error) {
	select {
	case <-pm.ready:
	default:
		return "", ErrNotReady
	}

	pm.m.RLock()
	defer pm.m.RUnlock()

	return pm.hostname, pm.err
}

// Close unsubscribes to the local LiteFS node's event stream.
func (pm *PrimaryMonitor) Close() {
	pm.es.Close()
	pm.setError(ErrClosed)
}

func (pm *PrimaryMonitor) run() {
	first := true

	for {
		select {
		case event, running := <-pm.es.C():
			if !running {
				return
			}
			switch data := event.Data.(type) {
			case *InitEventData:
				pm.setData(data.IsPrimary, data.Hostname)
			case *PrimaryChangeEventData:
				pm.setData(data.IsPrimary, data.Hostname)
			}
		case err, running := <-pm.es.ErrC():
			if !running {
				return
			}
			pm.setError(err)
		}

		if first {
			close(pm.ready)
			first = false
		}
	}
}

func (pm *PrimaryMonitor) setData(isPrimary bool, hostname string) {
	pm.m.Lock()
	defer pm.m.Unlock()

	pm.isPrimary = isPrimary
	pm.hostname = hostname
	pm.err = nil
}

func (pm *PrimaryMonitor) setError(err error) {
	pm.m.Lock()
	defer pm.m.Unlock()

	pm.err = err
}
