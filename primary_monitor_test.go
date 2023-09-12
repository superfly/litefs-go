package litefs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPrimaryMonitor(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		pm, c := mockServerMonitor(t)

		c <- initEventJSON
		c <- flush
		assertReady(t, pm, 5*time.Millisecond)
		assertPrimary(t, pm, true, "node-1")

		c <- pChangeNode2EventJSON
		c <- flush
		assertPrimary(t, pm, false, "node-2")

		c <- pChangeNode1EventJSON
		c <- flush
		assertPrimary(t, pm, true, "node-1")
	})

	t.Run("server error", func(t *testing.T) {
		pm, c := mockServerMonitor(t)

		c <- initEventJSON
		c <- flush
		assertReady(t, pm, 5*time.Millisecond)
		assertPrimary(t, pm, true, "node-1")

		c <- hangup
		c <- status500
		time.Sleep(5 * time.Millisecond)

		hn, err := pm.Hostname()
		if !errors.Is(err, errUnexpectedStatus) {
			t.Fatalf("expected errUnexpectedStatus, got %v", err)
		}
		if hn != "node-1" {
			t.Fatalf("expected node-1, got %s", hn)
		}

		ip, err := pm.IsPrimary()
		if !errors.Is(err, errUnexpectedStatus) {
			t.Fatalf("expected errUnexpectedStatus, got %v", err)
		}
		if !ip {
			t.Fatal("expected isPrimary")
		}

		c <- initEventJSON
		c <- flush
		assertPrimary(t, pm, true, "node-1")
	})

	t.Run("WaitReady timeout", func(t *testing.T) {
		pm, c := mockServerMonitor(t)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := pm.WaitReady(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected DeadlineExceeded, got %v", err)
		}

		c <- initEventJSON
		c <- flush
		assertReady(t, pm, 5*time.Millisecond)
		assertPrimary(t, pm, true, "node-1")
	})

	t.Run("WaitReady server error", func(t *testing.T) {
		pm, c := mockServerMonitor(t)

		c <- status500

		err := pm.WaitReady(context.Background())
		if !errors.Is(err, errUnexpectedStatus) {
			t.Fatalf("expected errUnexpectedStatus, got %v", err)
		}

		c <- initEventJSON
		c <- flush
		assertPrimary(t, pm, true, "node-1")
	})
}

func assertReady(t *testing.T, pm *PrimaryMonitor, to time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), to)
	defer cancel()

	if err := pm.WaitReady(ctx); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func assertPrimary(t *testing.T, pm *PrimaryMonitor, expectedIP bool, expectedHN string) {
	t.Helper()
	time.Sleep(5 * time.Millisecond)

	hn, err := pm.Hostname()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if hn != expectedHN {
		t.Fatalf("expected %s, got %s", expectedHN, hn)
	}

	ip, err := pm.IsPrimary()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if ip != expectedIP {
		t.Fatalf("expected %t, got %t", expectedIP, ip)
	}
}

func mockServerMonitor(t *testing.T) (*PrimaryMonitor, chan string) {
	c := make(chan string)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Err() != nil {
			return
		}

		for resp := range c {
			switch resp {
			case status500:
				w.WriteHeader(http.StatusInternalServerError)
				return
			case hangup:
				conn, _, _ := w.(http.Hijacker).Hijack()
				conn.Close()
				return
			case sleep10:
				time.Sleep(10 * time.Millisecond)
			case flush:
				w.(http.Flusher).Flush()
			default:
				fmt.Fprintln(w, resp)
			}
		}
	}))
	t.Cleanup(s.Close)
	EventSubscriptionURL = s.URL

	pm := NewPrimaryMonitor()
	t.Cleanup(pm.Close)
	t.Cleanup(func() { close(c) })

	return pm, c
}
