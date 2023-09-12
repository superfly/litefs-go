package litefs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	EventSubscriptionClient = http.DefaultClient
	EventSubscriptionURL    = "http://localhost:20202/events"
)

var (
	errUnexpectedStatus = errors.New("unexpected status")
)

// EventSubscription tracks events published by a LiteFS node.
type EventSubscription struct {
	c     chan *Event
	errc  chan error
	ctx   context.Context
	close func()
}

func SubscribeEvents() *EventSubscription {
	ctx, close := context.WithCancel(context.Background())

	es := &EventSubscription{
		c:     make(chan *Event),
		errc:  make(chan error),
		ctx:   ctx,
		close: close,
	}

	go es.run()

	return es
}

func (es *EventSubscription) run() {
	defer close(es.c)
	defer close(es.errc)

	for {
		err := es.doRequest()
		if es.ctx.Err() != nil {
			return
		}

		es.errc <- err
	}
}

func (es *EventSubscription) doRequest() error {
	req, err := http.NewRequestWithContext(es.ctx, http.MethodGet, EventSubscriptionURL, nil)
	if err != nil {
		return err
	}

	resp, err := EventSubscriptionClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", errUnexpectedStatus, resp.StatusCode)
	}

	d := json.NewDecoder(resp.Body)
	for {
		var e Event
		if err := d.Decode(&e); err != nil {
			return err
		}

		es.c <- &e
	}
}

// C returns a chan of events from the local LiteFS node.
func (es *EventSubscription) C() <-chan *Event {
	return es.c
}

// ErrC returns a chan of errors encountered while fetching events from the
// local LiteFS node.
func (es *EventSubscription) ErrC() <-chan error {
	return es.errc
}

// Close shuts down the EventSubscription.
func (es *EventSubscription) Close() {
	es.close()
}
