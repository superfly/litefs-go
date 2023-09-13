package litefs

import (
	"errors"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Open file description lock constants.
const (
	F_OFD_GETLK  = 36
	F_OFD_SETLK  = 37
	F_OFD_SETLKW = 38
)

// LiteFS lock offsets
const (
	HaltByte = 72
)

// Halt locks the HALT lock on the file handle to the LiteFS database lock file.
// This causes writes to be halted on the primary node so that this replica can
// perform writes until Unhalt() is invoked.
//
// The HALT lock will automatically be released when the file descriptor is closed.
//
// This function is a no-op if the current node is the primary.
func Halt(f *os.File) error {
	for {
		if err := syscall.FcntlFlock(f.Fd(), F_OFD_SETLKW, &syscall.Flock_t{
			Type:  syscall.F_WRLCK,
			Start: HaltByte,
			Len:   1,
		}); err != syscall.EINTR {
			return err // exit on non-EINTR error
		}
	}
}

// Unhalt releases the HALT lock on the file handle to the LiteFS database lock
// file. This allows writes to resume on the primary node. This replica will
// not be able to perform any more writes until Halt() is called again.
//
// This function is a no-op if the current node is the primary or if the
// lock expired.
func Unhalt(f *os.File) error {
	for {
		if err := syscall.FcntlFlock(f.Fd(), F_OFD_SETLKW, &syscall.Flock_t{
			Type:  syscall.F_UNLCK,
			Start: HaltByte,
			Len:   1,
		}); err != syscall.EINTR {
			return err // exit on non-EINTR error
		}
	}
}

// WithHalt executes fn with a HALT lock. This allows any node to perform writes
// on the database file. If this is a replica node, it will forward those writes
// back to the primary node. If this is the primary node, it will simply execute
// fn since it does not need the HALT lock.
//
// Please note that this will stop all writes on the primary. It is slower than
// processing a write directly on the primary since it requires two round trips
// to acquire & release the lock.
//
// This function should only be used for periodic migrations or low-write
// scenarios.
func WithHalt(databasePath string, fn func() error) error {
	f, err := os.OpenFile(databasePath+"-lock", os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := Halt(f); err != nil {
		return err
	}

	if err := fn(); err != nil {
		return err
	}

	return Unhalt(f)
}

var errNotReplicated = errors.New("initial replication from primary not finished yet")

// Lag reports the how far this node is lagging behind the primary. The lag is
// 0 on the primary node. In the absence of new transactions from the primary,
// heartbeat messages are sent at one second intervals.
func Lag(databasePath string) (time.Duration, error) {
	content, err := os.ReadFile(path.Dir(databasePath) + "/.lag")
	if err != nil {
		return 0, err
	}

	lag, err := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 32)
	if err != nil {
		return 0, err
	}

	if lag == math.MaxInt32 {
		return 0, errNotReplicated
	}

	return time.Duration(lag) * time.Millisecond, nil
}
