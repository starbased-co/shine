package state

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPrismEntrySetGetName(t *testing.T) {
	entry := &PrismEntry{}

	// Test normal name
	entry.SetName("clock")
	if got := entry.GetName(); got != "clock" {
		t.Errorf("GetName() = %q, want %q", got, "clock")
	}

	// Test max length name
	longName := make([]byte, 100)
	for i := range longName {
		longName[i] = 'a'
	}
	entry.SetName(string(longName))
	if got := entry.GetName(); len(got) != NameMaxLen {
		t.Errorf("GetName() length = %d, want %d", len(got), NameMaxLen)
	}

	// Test empty name
	entry.SetName("")
	if got := entry.GetName(); got != "" {
		t.Errorf("GetName() = %q, want empty string", got)
	}
}

func TestPrismEntryState(t *testing.T) {
	entry := &PrismEntry{State: uint8(PrismStateFg)}
	if got := entry.GetState(); got != PrismStateFg {
		t.Errorf("GetState() = %v, want %v", got, PrismStateFg)
	}

	entry.State = uint8(PrismStateBg)
	if got := entry.GetState(); got != PrismStateBg {
		t.Errorf("GetState() = %v, want %v", got, PrismStateBg)
	}
}

func TestPrismEntryUptime(t *testing.T) {
	entry := &PrismEntry{}

	// Zero start time
	if got := entry.Uptime(); got != 0 {
		t.Errorf("Uptime() = %v, want 0", got)
	}

	// Set start time
	entry.StartMs = time.Now().Add(-time.Second).UnixMilli()
	uptime := entry.Uptime()
	if uptime < 900*time.Millisecond || uptime > 1100*time.Millisecond {
		t.Errorf("Uptime() = %v, want ~1s", uptime)
	}
}

func TestPrismEntryIsActive(t *testing.T) {
	entry := &PrismEntry{}
	if entry.IsActive() {
		t.Error("IsActive() = true, want false for zero PID")
	}

	entry.PID = 1234
	if !entry.IsActive() {
		t.Error("IsActive() = false, want true for non-zero PID")
	}
}

func TestPrismRuntimeStateInstance(t *testing.T) {
	state := &PrismRuntimeState{}

	state.SetInstance("panel-0")
	if got := state.GetInstance(); got != "panel-0" {
		t.Errorf("GetInstance() = %q, want %q", got, "panel-0")
	}
}

func TestPrismRuntimeStateFgPrism(t *testing.T) {
	state := &PrismRuntimeState{}

	state.SetFgPrism("clock")
	if got := state.GetFgPrism(); got != "clock" {
		t.Errorf("GetFgPrism() = %q, want %q", got, "clock")
	}
}

func TestPrismStateWriterReader(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test.state")

	// Create writer
	writer, err := NewPrismStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewPrismStateWriter() error: %v", err)
	}
	defer writer.Remove()

	// Set instance
	writer.SetInstance("panel-test")

	// Add prisms
	idx, err := writer.AddPrism("clock", 1001, true)
	if err != nil {
		t.Fatalf("AddPrism() error: %v", err)
	}
	if idx != 0 {
		t.Errorf("AddPrism() index = %d, want 0", idx)
	}

	idx, err = writer.AddPrism("bar", 1002, false)
	if err != nil {
		t.Fatalf("AddPrism() error: %v", err)
	}
	if idx != 1 {
		t.Errorf("AddPrism() index = %d, want 1", idx)
	}

	// Open reader
	reader, err := OpenPrismStateReader(statePath)
	if err != nil {
		t.Fatalf("OpenPrismStateReader() error: %v", err)
	}
	defer reader.Close()

	// Read state
	state, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if got := state.GetInstance(); got != "panel-test" {
		t.Errorf("instance = %q, want %q", got, "panel-test")
	}

	if got := state.GetFgPrism(); got != "clock" {
		t.Errorf("fg prism = %q, want %q", got, "clock")
	}

	if state.PrismCount != 2 {
		t.Errorf("PrismCount = %d, want 2", state.PrismCount)
	}

	// Check prism entries
	prisms := state.ActivePrisms()
	if len(prisms) != 2 {
		t.Errorf("ActivePrisms() len = %d, want 2", len(prisms))
	}

	if prisms[0].GetName() != "clock" {
		t.Errorf("prism[0].name = %q, want %q", prisms[0].GetName(), "clock")
	}
	if prisms[0].PID != 1001 {
		t.Errorf("prism[0].PID = %d, want 1001", prisms[0].PID)
	}
	if prisms[0].GetState() != PrismStateFg {
		t.Errorf("prism[0].state = %v, want fg", prisms[0].GetState())
	}

	if prisms[1].GetName() != "bar" {
		t.Errorf("prism[1].name = %q, want %q", prisms[1].GetName(), "bar")
	}
	if prisms[1].GetState() != PrismStateBg {
		t.Errorf("prism[1].state = %v, want bg", prisms[1].GetState())
	}
}

func TestPrismStateWriterSetForeground(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test.state")

	writer, err := NewPrismStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewPrismStateWriter() error: %v", err)
	}
	defer writer.Remove()

	writer.AddPrism("clock", 1001, true)
	writer.AddPrism("bar", 1002, false)

	// Switch foreground
	writer.SetForeground("bar")

	reader, err := OpenPrismStateReader(statePath)
	if err != nil {
		t.Fatalf("OpenPrismStateReader() error: %v", err)
	}
	defer reader.Close()

	state, _ := reader.Read()

	if got := state.GetFgPrism(); got != "bar" {
		t.Errorf("fg prism = %q, want %q", got, "bar")
	}

	// Check state flags
	prisms := state.ActivePrisms()
	for _, p := range prisms {
		if p.GetName() == "bar" && p.GetState() != PrismStateFg {
			t.Errorf("bar should be fg")
		}
		if p.GetName() == "clock" && p.GetState() != PrismStateBg {
			t.Errorf("clock should be bg")
		}
	}
}

func TestPrismStateWriterRemovePrism(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test.state")

	writer, err := NewPrismStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewPrismStateWriter() error: %v", err)
	}
	defer writer.Remove()

	writer.AddPrism("clock", 1001, true)
	writer.AddPrism("bar", 1002, false)
	writer.AddPrism("chat", 1003, false)

	// Remove middle prism
	writer.RemovePrism("bar")

	reader, err := OpenPrismStateReader(statePath)
	if err != nil {
		t.Fatalf("OpenPrismStateReader() error: %v", err)
	}
	defer reader.Close()

	state, _ := reader.Read()

	if state.PrismCount != 2 {
		t.Errorf("PrismCount = %d, want 2", state.PrismCount)
	}

	prisms := state.ActivePrisms()
	names := make([]string, len(prisms))
	for i, p := range prisms {
		names[i] = p.GetName()
	}

	// Should have clock and chat, not bar
	if names[0] != "clock" || names[1] != "chat" {
		t.Errorf("prisms = %v, want [clock, chat]", names)
	}
}

func TestConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test.state")

	writer, err := NewPrismStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewPrismStateWriter() error: %v", err)
	}
	defer writer.Remove()

	writer.SetInstance("panel-test")
	writer.AddPrism("clock", 1001, true)

	// Start concurrent readers
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reader, err := OpenPrismStateReader(statePath)
			if err != nil {
				errors <- err
				return
			}
			defer reader.Close()

			for j := 0; j < 100; j++ {
				state, err := reader.Read()
				if err != nil {
					errors <- err
					return
				}
				if state.GetInstance() != "panel-test" {
					errors <- err
					return
				}
			}
		}()
	}

	// Writer updates concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			writer.SetForeground("clock")
		}
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent read error: %v", err)
	}
}

func TestSequenceCounter(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "test.state")

	writer, err := NewPrismStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewPrismStateWriter() error: %v", err)
	}
	defer writer.Remove()

	reader, err := OpenPrismStateReader(statePath)
	if err != nil {
		t.Fatalf("OpenPrismStateReader() error: %v", err)
	}
	defer reader.Close()

	// Initial version should be 0 (even = complete)
	v := reader.Version()
	if v != 0 {
		t.Errorf("initial version = %d, want 0", v)
	}
	if reader.IsWriting() {
		t.Error("IsWriting() = true, want false initially")
	}

	// After write, version should be even and > 0
	writer.SetInstance("test")
	v = reader.Version()
	if v == 0 {
		t.Error("version should be > 0 after write")
	}
	if v%2 != 0 {
		t.Errorf("version = %d, should be even", v)
	}
}

func TestMappedFileFileSizeMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "bad.state")

	// Create a file with wrong size
	f, _ := os.Create(statePath)
	f.Write([]byte("small"))
	f.Close()

	_, err := OpenMappedFile(statePath, PrismRuntimeStateSize)
	if err == nil {
		t.Error("expected error for file size mismatch")
	}
}

func TestShinedStateWriterReader(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "shined.state")

	writer, err := NewShinedStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewShinedStateWriter() error: %v", err)
	}
	defer writer.Remove()

	// Add panels
	idx, err := writer.AddPanel("panel-0", "main", 2001, true)
	if err != nil {
		t.Fatalf("AddPanel() error: %v", err)
	}
	if idx != 0 {
		t.Errorf("AddPanel() index = %d, want 0", idx)
	}

	writer.AddPanel("panel-1", "side", 2002, false)

	// Read
	reader, err := OpenShinedStateReader(statePath)
	if err != nil {
		t.Fatalf("OpenShinedStateReader() error: %v", err)
	}
	defer reader.Close()

	state, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if state.PanelCount != 2 {
		t.Errorf("PanelCount = %d, want 2", state.PanelCount)
	}

	panels := state.ActivePanels()
	if len(panels) != 2 {
		t.Errorf("ActivePanels() len = %d, want 2", len(panels))
	}

	if panels[0].GetInstance() != "panel-0" {
		t.Errorf("panel[0].instance = %q, want %q", panels[0].GetInstance(), "panel-0")
	}
	if !panels[0].IsHealthy() {
		t.Error("panel[0] should be healthy")
	}
	if panels[1].IsHealthy() {
		t.Error("panel[1] should not be healthy")
	}
}

func TestShinedStateWriterSetHealth(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "shined.state")

	writer, err := NewShinedStateWriter(statePath)
	if err != nil {
		t.Fatalf("NewShinedStateWriter() error: %v", err)
	}
	defer writer.Remove()

	writer.AddPanel("panel-0", "main", 2001, false)
	writer.SetPanelHealth("panel-0", true)

	reader, err := OpenShinedStateReader(statePath)
	if err != nil {
		t.Fatalf("OpenShinedStateReader() error: %v", err)
	}
	defer reader.Close()

	state, _ := reader.Read()
	panels := state.ActivePanels()
	if !panels[0].IsHealthy() {
		t.Error("panel should be healthy after SetPanelHealth(true)")
	}
}

func TestStructSizes(t *testing.T) {
	// These are verified at init() but test them explicitly
	tests := []struct {
		name string
		size int
		want int
	}{
		{"PrismEntry", int(PrismEntrySize), 80},
		{"PrismRuntimeState", int(PrismRuntimeStateSize), 1424},
		{"PanelEntry", int(PanelEntrySize), 136},
		{"ShinedState", int(ShinedStateSize), 4368},
	}

	for _, tt := range tests {
		if tt.size != tt.want {
			t.Errorf("%s size = %d, want %d", tt.name, tt.size, tt.want)
		}
	}
}
