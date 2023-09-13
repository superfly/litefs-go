package litefs

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"
)

func TestLag(t *testing.T) {
	const lagFmt = "%+010d\n"

	var (
		dbDir   = t.TempDir()
		dbPath  = dbDir + "/foo.db"
		lagPath = dbDir + "/.lag"
	)

	if _, err := Lag(dbPath); err == nil {
		t.Fatal("expected error")
	}

	os.WriteFile(lagPath, []byte("hi"), 0o666)
	if _, err := Lag(dbPath); err == nil {
		t.Fatal("expected error")
	}

	os.WriteFile(lagPath, []byte(fmt.Sprintf(lagFmt, math.MaxInt32)), 0o666)
	if _, err := Lag(dbPath); err != errNotReplicated {
		t.Fatalf("expected errNotReplicated, got %v", err)
	}

	os.WriteFile(lagPath, []byte(fmt.Sprintf(lagFmt, 0)), 0o666)
	if lag, err := Lag(dbPath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if lag != 0 {
		t.Fatalf("expected 0ms, got %v", lag)
	}

	os.WriteFile(lagPath, []byte(fmt.Sprintf(lagFmt, 123)), 0o666)
	if lag, err := Lag(dbPath); err != nil {
		t.Fatalf("expected no error, got %v", err)
	} else if lag != 123*time.Millisecond {
		t.Fatalf("expected 0ms, got %v", lag)
	}
}
