package state

import (
	"fmt"
	"time"
	"unsafe"
)

// Compile-time size verification
const (
	PrismEntrySize        = 80   // bytes per prism entry
	MaxPrisms             = 16   // max prisms per prismctl instance
	PrismRuntimeStateSize = 1424 // total size of PrismRuntimeState (includes alignment padding)

	PanelEntrySize       = 136  // bytes per panel entry
	MaxPanels            = 32   // max panels
	ShinedStateSize    = 4368 // total size of ShinedState

	NameMaxLen = 63 // max length for names (64 bytes with length prefix)
)

// PrismEntryState represents the state of a prism
type PrismEntryState uint8

const (
	PrismStateBg PrismEntryState = 0 // background
	PrismStateFg PrismEntryState = 1 // foreground
)

func (s PrismEntryState) String() string {
	switch s {
	case PrismStateBg:
		return "bg"
	case PrismStateFg:
		return "fg"
	default:
		return "unknown"
	}
}

// PrismEntry is a fixed-size entry for a single prism
// Total size: 80 bytes
type PrismEntry struct {
	NameLen  uint8     // 1 byte: length of name
	Name     [63]byte  // 63 bytes: name (null-padded)
	PID      int32     // 4 bytes: process ID
	State    uint8     // 1 byte: 0=bg, 1=fg
	Restarts uint8     // 1 byte: restart count (capped at 255)
	_padding [2]byte   // 2 bytes: padding for alignment
	StartMs  int64     // 8 bytes: unix ms when started
}

// GetName returns the prism name as a string
func (e *PrismEntry) GetName() string {
	return string(e.Name[:e.NameLen])
}

// SetName sets the prism name
func (e *PrismEntry) SetName(name string) {
	if len(name) > NameMaxLen {
		name = name[:NameMaxLen]
	}
	e.NameLen = uint8(len(name))
	copy(e.Name[:], name)
	// Zero rest of buffer
	for i := int(e.NameLen); i < NameMaxLen; i++ {
		e.Name[i] = 0
	}
}

// GetState returns the prism state
func (e *PrismEntry) GetState() PrismEntryState {
	return PrismEntryState(e.State)
}

// Uptime returns the duration since the prism started
func (e *PrismEntry) Uptime() time.Duration {
	if e.StartMs == 0 {
		return 0
	}
	return time.Duration(time.Now().UnixMilli()-e.StartMs) * time.Millisecond
}

// IsActive returns true if the entry is in use
func (e *PrismEntry) IsActive() bool {
	return e.PID != 0
}

// PrismRuntimeState is the mmap-friendly state structure for prismctl
// Total size: 1344 bytes
type PrismRuntimeState struct {
	Version     uint64           // 8 bytes: sequence counter (odd=writing, even=complete)
	InstanceLen uint8            // 1 byte: length of instance name
	Instance    [63]byte         // 63 bytes: instance name
	FgPrismLen  uint8            // 1 byte: length of foreground prism name
	FgPrism     [63]byte         // 63 bytes: foreground prism name
	PrismCount  uint8            // 1 byte: number of active prisms
	_padding    [3]byte          // 3 bytes: padding for alignment
	Prisms      [16]PrismEntry   // 16 * 80 = 1280 bytes
}

// GetInstance returns the instance name
func (s *PrismRuntimeState) GetInstance() string {
	return string(s.Instance[:s.InstanceLen])
}

// SetInstance sets the instance name
func (s *PrismRuntimeState) SetInstance(name string) {
	if len(name) > NameMaxLen {
		name = name[:NameMaxLen]
	}
	s.InstanceLen = uint8(len(name))
	copy(s.Instance[:], name)
	for i := int(s.InstanceLen); i < NameMaxLen; i++ {
		s.Instance[i] = 0
	}
}

// GetFgPrism returns the foreground prism name
func (s *PrismRuntimeState) GetFgPrism() string {
	return string(s.FgPrism[:s.FgPrismLen])
}

// SetFgPrism sets the foreground prism name
func (s *PrismRuntimeState) SetFgPrism(name string) {
	if len(name) > NameMaxLen {
		name = name[:NameMaxLen]
	}
	s.FgPrismLen = uint8(len(name))
	copy(s.FgPrism[:], name)
	for i := int(s.FgPrismLen); i < NameMaxLen; i++ {
		s.FgPrism[i] = 0
	}
}

// ActivePrisms returns a slice of active prism entries
func (s *PrismRuntimeState) ActivePrisms() []PrismEntry {
	result := make([]PrismEntry, 0, s.PrismCount)
	for i := 0; i < int(s.PrismCount); i++ {
		if s.Prisms[i].IsActive() {
			result = append(result, s.Prisms[i])
		}
	}
	return result
}

// PanelEntry is a fixed-size entry for a single panel
// Total size: 136 bytes
type PanelEntry struct {
	InstanceLen uint8     // 1 byte: length of instance name
	Instance    [63]byte  // 63 bytes: instance name
	NameLen     uint8     // 1 byte: length of panel name
	Name        [63]byte  // 63 bytes: panel name
	PID         int32     // 4 bytes: prismctl process ID
	Healthy     uint8     // 1 byte: health status (0=unhealthy, 1=healthy)
	_padding    [3]byte   // 3 bytes: padding for alignment
}

// GetInstance returns the panel instance name
func (e *PanelEntry) GetInstance() string {
	return string(e.Instance[:e.InstanceLen])
}

// SetInstance sets the panel instance name
func (e *PanelEntry) SetInstance(name string) {
	if len(name) > NameMaxLen {
		name = name[:NameMaxLen]
	}
	e.InstanceLen = uint8(len(name))
	copy(e.Instance[:], name)
	for i := int(e.InstanceLen); i < NameMaxLen; i++ {
		e.Instance[i] = 0
	}
}

// GetName returns the panel name
func (e *PanelEntry) GetName() string {
	return string(e.Name[:e.NameLen])
}

// SetName sets the panel name
func (e *PanelEntry) SetName(name string) {
	if len(name) > NameMaxLen {
		name = name[:NameMaxLen]
	}
	e.NameLen = uint8(len(name))
	copy(e.Name[:], name)
	for i := int(e.NameLen); i < NameMaxLen; i++ {
		e.Name[i] = 0
	}
}

// IsHealthy returns the health status
func (e *PanelEntry) IsHealthy() bool {
	return e.Healthy == 1
}

// IsActive returns true if the entry is in use
func (e *PanelEntry) IsActive() bool {
	return e.PID != 0
}

// ShinedState is the mmap-friendly state structure for shined
// Total size: 4368 bytes
type ShinedState struct {
	Version    uint64           // 8 bytes: sequence counter (odd=writing, even=complete)
	PanelCount uint8            // 1 byte: number of active panels
	_padding   [7]byte          // 7 bytes: padding for alignment
	Panels     [32]PanelEntry   // 32 * 136 = 4352 bytes
}

// ActivePanels returns a slice of active panel entries
func (s *ShinedState) ActivePanels() []PanelEntry {
	result := make([]PanelEntry, 0, s.PanelCount)
	for i := 0; i < int(s.PanelCount); i++ {
		if s.Panels[i].IsActive() {
			result = append(result, s.Panels[i])
		}
	}
	return result
}

func init() {
	// Verify struct sizes at compile time
	if size := unsafe.Sizeof(PrismEntry{}); size != PrismEntrySize {
		panic(fmt.Sprintf("PrismEntry size mismatch: got %d, want %d", size, PrismEntrySize))
	}
	if size := unsafe.Sizeof(PrismRuntimeState{}); size != PrismRuntimeStateSize {
		panic(fmt.Sprintf("PrismRuntimeState size mismatch: got %d, want %d", size, PrismRuntimeStateSize))
	}
	if size := unsafe.Sizeof(PanelEntry{}); size != PanelEntrySize {
		panic(fmt.Sprintf("PanelEntry size mismatch: got %d, want %d", size, PanelEntrySize))
	}
	if size := unsafe.Sizeof(ShinedState{}); size != ShinedStateSize {
		panic(fmt.Sprintf("ShinedState size mismatch: got %d, want %d", size, ShinedStateSize))
	}
}
