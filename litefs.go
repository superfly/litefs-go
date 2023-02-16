package litefs

import (
	"os"
	"syscall"
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

// Halt locks the HALT lock on the file handle to the SQLite database. This
// causes writes to be halted on the primary node so that this replica can
// perform writes until Unhalt() is invoked.
//
// The HALT lock will automatically be released when the file descriptor is closed.
//
// This function is a no-op if the current node is the primary.
func Halt(f *os.File) error {
	return syscall.FcntlFlock(f.Fd(), F_OFD_SETLKW, &syscall.Flock_t{
		Type:  syscall.F_WRLCK,
		Start: HaltByte,
		Len:   1,
	})
}

// Unhalt releases the HALT lock on the file handle to the SQLite database.
// allows writes to resume on the primary node. This replica will not be able
// to perform any more writes until Halt() is called again.
//
// This function is a no-op if the current node is the primary or if the
// lock expired.
func Unhalt(f *os.File) error {
	return syscall.FcntlFlock(f.Fd(), F_OFD_SETLKW, &syscall.Flock_t{
		Type:  syscall.F_UNLCK,
		Start: HaltByte,
		Len:   1,
	})
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
func WithHalt(path string, fn func() error) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0666)
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
