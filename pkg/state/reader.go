package state

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// MaxReadRetries is the maximum number of retries for consistent reads
const MaxReadRetries = 10

// PrismStateReader reads prism state from an mmap file
type PrismStateReader struct {
	mmap *MappedFile
	ptr  *PrismRuntimeState
}

// OpenPrismStateReader opens an existing state file for reading
func OpenPrismStateReader(path string) (*PrismStateReader, error) {
	mmap, err := OpenMappedFile(path, PrismRuntimeStateSize)
	if err != nil {
		return nil, err
	}

	ptr := (*PrismRuntimeState)(unsafe.Pointer(&mmap.Data()[0]))

	return &PrismStateReader{
		mmap: mmap,
		ptr:  ptr,
	}, nil
}

// Read performs a consistent read of the state
// Returns an error if a consistent read cannot be achieved after MaxReadRetries
func (r *PrismStateReader) Read() (*PrismRuntimeState, error) {
	for i := 0; i < MaxReadRetries; i++ {
		// Read version before
		v1 := atomic.LoadUint64(&r.ptr.Version)

		// If version is odd, writer is in progress - retry
		if v1%2 != 0 {
			continue
		}

		// Copy the state
		state := *r.ptr

		// Read version after
		v2 := atomic.LoadUint64(&r.ptr.Version)

		// If versions match and even, we have a consistent read
		if v1 == v2 {
			return &state, nil
		}
	}

	return nil, fmt.Errorf("failed to get consistent read after %d retries", MaxReadRetries)
}

// ReadFast reads state without consistency check (for polling)
// Caller should verify version is even for valid state
func (r *PrismStateReader) ReadFast() (*PrismRuntimeState, uint64) {
	state := *r.ptr
	return &state, atomic.LoadUint64(&r.ptr.Version)
}

// Version returns the current version
func (r *PrismStateReader) Version() uint64 {
	return atomic.LoadUint64(&r.ptr.Version)
}

// IsWriting returns true if a write is in progress (version is odd)
func (r *PrismStateReader) IsWriting() bool {
	return r.Version()%2 != 0
}

// Close closes the reader
func (r *PrismStateReader) Close() error {
	return r.mmap.Close()
}

// Path returns the state file path
func (r *PrismStateReader) Path() string {
	return r.mmap.Path()
}

// ShinedStateReader reads shined state from an mmap file
type ShinedStateReader struct {
	mmap *MappedFile
	ptr  *ShinedState
}

// OpenShinedStateReader opens an existing state file for reading
func OpenShinedStateReader(path string) (*ShinedStateReader, error) {
	mmap, err := OpenMappedFile(path, ShinedStateSize)
	if err != nil {
		return nil, err
	}

	ptr := (*ShinedState)(unsafe.Pointer(&mmap.Data()[0]))

	return &ShinedStateReader{
		mmap: mmap,
		ptr:  ptr,
	}, nil
}

// Read performs a consistent read of the state
func (r *ShinedStateReader) Read() (*ShinedState, error) {
	for i := 0; i < MaxReadRetries; i++ {
		v1 := atomic.LoadUint64(&r.ptr.Version)
		if v1%2 != 0 {
			continue
		}

		state := *r.ptr

		v2 := atomic.LoadUint64(&r.ptr.Version)
		if v1 == v2 {
			return &state, nil
		}
	}

	return nil, fmt.Errorf("failed to get consistent read after %d retries", MaxReadRetries)
}

// ReadFast reads state without consistency check
func (r *ShinedStateReader) ReadFast() (*ShinedState, uint64) {
	state := *r.ptr
	return &state, atomic.LoadUint64(&r.ptr.Version)
}

// Version returns the current version
func (r *ShinedStateReader) Version() uint64 {
	return atomic.LoadUint64(&r.ptr.Version)
}

// IsWriting returns true if a write is in progress
func (r *ShinedStateReader) IsWriting() bool {
	return r.Version()%2 != 0
}

// Close closes the reader
func (r *ShinedStateReader) Close() error {
	return r.mmap.Close()
}

// Path returns the state file path
func (r *ShinedStateReader) Path() string {
	return r.mmap.Path()
}
