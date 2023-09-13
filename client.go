package litefs

import (
	"context"
	"net/http"
)

// SubscribeEvents subscribes to events from the default (localhost:20202)
// LiteFS node.
func SubscribeEvents() *EventSubscription {
	return DefaultClient.SubscribeEvents()
}

// MonitorPrimary monitors the primary status of the LiteFS cluster via the
// default (localhost:20202) LiteFS node's event stream.
func MonitorPrimary() *PrimaryMonitor {
	return DefaultClient.MonitorPrimary()
}

// Client is an HTTP client for communicating with a LiteFS node.
type Client struct {
	// Base URL of the LiteFS cluster node.
	URL string

	// HTTP client to use for requests to cluster node.
	HTTP *http.Client
}

// DefaultClient is a client for communicating with the default
// (localhost:20202) LiteFS node.
var DefaultClient = &Client{
	URL:  "http://localhost:20202",
	HTTP: http.DefaultClient,
}

// SubscribeEvents subscribes to events from the LiteFS node.
func (c *Client) SubscribeEvents() *EventSubscription {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventSubscription{
		c:         c,
		ctx:       ctx,
		cancelCtx: cancel,
	}
}

// MonitorPrimary monitors the primary status of the LiteFS cluster via the
// LiteFS node's event stream.
func (c *Client) MonitorPrimary() *PrimaryMonitor {
	pm := &PrimaryMonitor{
		es:    c.SubscribeEvents(),
		ready: make(chan struct{}),
	}

	go pm.run()

	return pm
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.HTTP.Do(req)
}
