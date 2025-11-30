package state

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// PrismStateWriter writes prism state to an mmap file with sequence counting
type PrismStateWriter struct {
	mu   sync.Mutex
	mmap *MappedFile
	ptr  *PrismRuntimeState
}

// NewPrismStateWriter creates a new state writer
func NewPrismStateWriter(path string) (*PrismStateWriter, error) {
	mmap, err := CreateMappedFile(path, PrismRuntimeStateSize)
	if err != nil {
		return nil, err
	}

	ptr := (*PrismRuntimeState)(mmap.AsPtr())
	// Initialize version to 0 (even = complete state)
	atomic.StoreUint64(&ptr.Version, 0)

	return &PrismStateWriter{
		mmap: mmap,
		ptr:  ptr,
	}, nil
}

// SetInstance sets the instance name
func (w *PrismStateWriter) SetInstance(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()
	w.ptr.SetInstance(name)
	w.endWrite()
}

// Update atomically updates the state
func (w *PrismStateWriter) Update(fn func(*PrismRuntimeState)) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()
	fn(w.ptr)
	w.endWrite()
}

// SetPrism updates a prism entry
func (w *PrismStateWriter) SetPrism(index int, name string, pid int32, state PrismEntryState, restarts uint8, startMs int64) error {
	if index < 0 || index >= MaxPrisms {
		return fmt.Errorf("prism index out of range: %d", index)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()

	entry := &w.ptr.Prisms[index]
	entry.SetName(name)
	entry.PID = pid
	entry.State = uint8(state)
	entry.Restarts = restarts
	entry.StartMs = startMs

	// Update count if this is a new entry
	if index >= int(w.ptr.PrismCount) {
		w.ptr.PrismCount = uint8(index + 1)
	}

	w.endWrite()
	return nil
}

// RemovePrism clears a prism entry and compacts the array
func (w *PrismStateWriter) RemovePrism(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()

	// Find and remove the prism
	found := -1
	for i := 0; i < int(w.ptr.PrismCount); i++ {
		if w.ptr.Prisms[i].GetName() == name {
			found = i
			break
		}
	}

	if found >= 0 {
		// Shift remaining entries down
		for i := found; i < int(w.ptr.PrismCount)-1; i++ {
			w.ptr.Prisms[i] = w.ptr.Prisms[i+1]
		}
		// Clear last entry
		w.ptr.Prisms[w.ptr.PrismCount-1] = PrismEntry{}
		w.ptr.PrismCount--
	}

	w.endWrite()
}

// SetForeground sets the foreground prism
func (w *PrismStateWriter) SetForeground(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()

	w.ptr.SetFgPrism(name)

	// Update state flags
	for i := 0; i < int(w.ptr.PrismCount); i++ {
		if w.ptr.Prisms[i].GetName() == name {
			w.ptr.Prisms[i].State = uint8(PrismStateFg)
		} else {
			w.ptr.Prisms[i].State = uint8(PrismStateBg)
		}
	}

	w.endWrite()
}

// AddPrism adds a new prism and returns its index
func (w *PrismStateWriter) AddPrism(name string, pid int32, fg bool) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.ptr.PrismCount >= MaxPrisms {
		return -1, fmt.Errorf("max prisms reached: %d", MaxPrisms)
	}

	w.beginWrite()

	idx := int(w.ptr.PrismCount)
	entry := &w.ptr.Prisms[idx]
	entry.SetName(name)
	entry.PID = pid
	if fg {
		entry.State = uint8(PrismStateFg)
		w.ptr.SetFgPrism(name)
		// Set all others to background
		for i := 0; i < idx; i++ {
			w.ptr.Prisms[i].State = uint8(PrismStateBg)
		}
	} else {
		entry.State = uint8(PrismStateBg)
	}
	entry.Restarts = 0
	entry.StartMs = time.Now().UnixMilli()
	w.ptr.PrismCount++

	w.endWrite()
	return idx, nil
}

// beginWrite increments version to odd (writing state)
func (w *PrismStateWriter) beginWrite() {
	v := atomic.LoadUint64(&w.ptr.Version)
	atomic.StoreUint64(&w.ptr.Version, v+1)
}

// endWrite increments version to even (complete state) and syncs
func (w *PrismStateWriter) endWrite() {
	v := atomic.LoadUint64(&w.ptr.Version)
	atomic.StoreUint64(&w.ptr.Version, v+1)
	w.mmap.Sync()
}

// Sync forces a sync to disk
func (w *PrismStateWriter) Sync() error {
	return w.mmap.Sync()
}

// Close closes the writer
func (w *PrismStateWriter) Close() error {
	return w.mmap.Close()
}

// Remove closes and removes the state file
func (w *PrismStateWriter) Remove() error {
	return w.mmap.Remove()
}

// Path returns the state file path
func (w *PrismStateWriter) Path() string {
	return w.mmap.Path()
}

// ShinedStateWriter writes shined state to an mmap file
type ShinedStateWriter struct {
	mu   sync.Mutex
	mmap *MappedFile
	ptr  *ShinedState
}

// NewShinedStateWriter creates a new shined state writer
func NewShinedStateWriter(path string) (*ShinedStateWriter, error) {
	mmap, err := CreateMappedFile(path, ShinedStateSize)
	if err != nil {
		return nil, err
	}

	ptr := (*ShinedState)(unsafe.Pointer(&mmap.Data()[0]))
	atomic.StoreUint64(&ptr.Version, 0)

	return &ShinedStateWriter{
		mmap: mmap,
		ptr:  ptr,
	}, nil
}

// Update atomically updates the state
func (w *ShinedStateWriter) Update(fn func(*ShinedState)) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()
	fn(w.ptr)
	w.endWrite()
}

// AddPanel adds a new panel
func (w *ShinedStateWriter) AddPanel(instance, name string, pid int32, healthy bool) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.ptr.PanelCount >= MaxPanels {
		return -1, fmt.Errorf("max panels reached: %d", MaxPanels)
	}

	w.beginWrite()

	idx := int(w.ptr.PanelCount)
	entry := &w.ptr.Panels[idx]
	entry.SetInstance(instance)
	entry.SetName(name)
	entry.PID = pid
	if healthy {
		entry.Healthy = 1
	} else {
		entry.Healthy = 0
	}
	w.ptr.PanelCount++

	w.endWrite()
	return idx, nil
}

// RemovePanel removes a panel by instance name
func (w *ShinedStateWriter) RemovePanel(instance string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()

	found := -1
	for i := 0; i < int(w.ptr.PanelCount); i++ {
		if w.ptr.Panels[i].GetInstance() == instance {
			found = i
			break
		}
	}

	if found >= 0 {
		for i := found; i < int(w.ptr.PanelCount)-1; i++ {
			w.ptr.Panels[i] = w.ptr.Panels[i+1]
		}
		w.ptr.Panels[w.ptr.PanelCount-1] = PanelEntry{}
		w.ptr.PanelCount--
	}

	w.endWrite()
}

// SetPanelHealth updates a panel's health status
func (w *ShinedStateWriter) SetPanelHealth(instance string, healthy bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.beginWrite()

	for i := 0; i < int(w.ptr.PanelCount); i++ {
		if w.ptr.Panels[i].GetInstance() == instance {
			if healthy {
				w.ptr.Panels[i].Healthy = 1
			} else {
				w.ptr.Panels[i].Healthy = 0
			}
			break
		}
	}

	w.endWrite()
}

func (w *ShinedStateWriter) beginWrite() {
	v := atomic.LoadUint64(&w.ptr.Version)
	atomic.StoreUint64(&w.ptr.Version, v+1)
}

func (w *ShinedStateWriter) endWrite() {
	v := atomic.LoadUint64(&w.ptr.Version)
	atomic.StoreUint64(&w.ptr.Version, v+1)
	w.mmap.Sync()
}

// Sync forces a sync to disk
func (w *ShinedStateWriter) Sync() error {
	return w.mmap.Sync()
}

// Close closes the writer
func (w *ShinedStateWriter) Close() error {
	return w.mmap.Close()
}

// Remove closes and removes the state file
func (w *ShinedStateWriter) Remove() error {
	return w.mmap.Remove()
}

// Path returns the state file path
func (w *ShinedStateWriter) Path() string {
	return w.mmap.Path()
}
