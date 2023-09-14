package litefs

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrNotReady = errors.New("awaiting first event")
)

// PrimaryMonitor monitors the current primary status of the LiteFS cluster.
type PrimaryMonitor interface {
	// WaitReady blocks until ctx expires or a response or error has been received
	// from the local LiteFS node. IsPrimary and Hostname will return errors until
	// this method returns nil.
	WaitReady(ctx context.Context) error

	// IsPrimary reports whether the local node is primary node in the cluster. An
	// error is returned if this method is called before data has been received
	// from the local LiteFS node (see WaitReady). If an error is encountered while
	// communicating with the node, that error, along with the most recent
	// IsPrimary value will be returned.
	IsPrimary() (bool, error)

	// Hostname reports the name of the current primary node in the cluster. An
	// error is returned if this method is called before data has been received
	// from the local LiteFS node (see WaitReady). If an error is encountered while
	// communicating with the node, that error, along with the most recent
	// Hostname value will be returned.
	Hostname() (string, error)

	// Close unsubscribes to the LiteFS node's event stream.
	Close()
}

type primaryMonitor struct {
	es    EventSubscription
	ready chan struct{}
	m     sync.RWMutex

	isPrimary bool
	hostname  string
	err       error
}

var _ PrimaryMonitor = (*primaryMonitor)(nil)

func (pm *primaryMonitor) WaitReady(ctx context.Context) error {
	select {
	case <-pm.ready:
		pm.m.RLock()
		defer pm.m.RUnlock()
		return pm.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pm *primaryMonitor) IsPrimary() (bool, error) {
	select {
	case <-pm.ready:
	default:
		return false, ErrNotReady
	}

	pm.m.RLock()
	defer pm.m.RUnlock()

	return pm.isPrimary, pm.err
}

func (pm *primaryMonitor) Hostname() (string, error) {
	select {
	case <-pm.ready:
	default:
		return "", ErrNotReady
	}

	pm.m.RLock()
	defer pm.m.RUnlock()

	return pm.hostname, pm.err
}

func (pm *primaryMonitor) Close() {
	pm.es.Close()
}

func (pm *primaryMonitor) run() {
	first := true

	for {
		if e, err := pm.es.Next(); err != nil {
			pm.setError(err)
		} else {
			switch data := e.Data.(type) {
			case *InitEventData:
				pm.setData(data.IsPrimary, data.Hostname)
			case *PrimaryChangeEventData:
				pm.setData(data.IsPrimary, data.Hostname)
			}
		}

		if first {
			close(pm.ready)
			first = false
		}
	}
}

func (pm *primaryMonitor) setData(isPrimary bool, hostname string) {
	pm.m.Lock()
	defer pm.m.Unlock()

	pm.isPrimary = isPrimary
	pm.hostname = hostname
	pm.err = nil
}

func (pm *primaryMonitor) setError(err error) {
	pm.m.Lock()
	defer pm.m.Unlock()

	pm.err = err
}
