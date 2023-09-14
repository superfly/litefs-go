package litefs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

var (
	ErrClosed           = errors.New("closed EventSubscription")
	errUnexpectedStatus = errors.New("unexpected status")
)

// EventSubscription monitors a LiteFS node for published events.
type EventSubscription interface {
	// Next attempts to read the next event from the LiteFS node. An error is
	// returned if the request fails. Calling `Next()` again after an error will
	// initiate a new HTTP request. ErrClosed is returned if the EventSubscription
	// is closed while this method is blocking.
	Next() (*Event, error)

	// Close aborts any in-progress requests to the LiteFS node.
	Close()
}

type eventSubscription struct {
	c         *Client
	ctx       context.Context
	d         *json.Decoder
	cancelCtx func()
	closeBody func() error
	m         sync.Mutex
}

var _ EventSubscription = (*eventSubscription)(nil)

func (es *eventSubscription) Next() (*Event, error) {
	e, err := es.next()
	if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrBodyReadAfterClose) {
		err = ErrClosed
	}
	return e, err
}

func (es *eventSubscription) next() (*Event, error) {
	es.m.Lock()
	defer es.m.Unlock()

	if es.d == nil {
		resp, err := es.c.get(es.ctx, "/events")
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("%w: %d", errUnexpectedStatus, resp.StatusCode)
		}

		es.closeBody = resp.Body.Close
		es.d = json.NewDecoder(resp.Body)
	}

	var e Event
	if err := es.d.Decode(&e); err != nil {
		es.closeBody()
		es.d = nil

		return nil, err
	}

	return &e, nil
}

func (es *eventSubscription) Close() {
	es.cancelCtx()

	es.m.Lock()
	defer es.m.Unlock()

	if es.d != nil {
		es.closeBody()
		es.d = nil
	}
}
