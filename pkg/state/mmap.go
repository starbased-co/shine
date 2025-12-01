package state

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type MappedFile struct {
	path string
	file *os.File
	data []byte
	size int
}

func OpenMappedFile(path string, size int) (*MappedFile, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}
	if info.Size() != int64(size) {
		file.Close()
		return nil, fmt.Errorf("file size mismatch: got %d, want %d", info.Size(), size)
	}

	data, err := unix.Mmap(int(file.Fd()), 0, size, unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to mmap %s: %w", path, err)
	}

	return &MappedFile{
		path: path,
		file: file,
		data: data,
		size: size,
	}, nil
}

func CreateMappedFile(path string, size int) (*MappedFile, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", path, err)
	}

	if err := file.Truncate(int64(size)); err != nil {
		file.Close()
		os.Remove(path)
		return nil, fmt.Errorf("failed to truncate %s: %w", path, err)
	}

	data, err := unix.Mmap(int(file.Fd()), 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		file.Close()
		os.Remove(path)
		return nil, fmt.Errorf("failed to mmap %s: %w", path, err)
	}

	return &MappedFile{
		path: path,
		file: file,
		data: data,
		size: size,
	}, nil
}

func (m *MappedFile) Data() []byte {
	return m.data
}

func (m *MappedFile) Path() string {
	return m.path
}

func (m *MappedFile) Size() int {
	return m.size
}

func (m *MappedFile) Sync() error {
	return unix.Msync(m.data, unix.MS_SYNC)
}

func (m *MappedFile) Close() error {
	var errs []error

	if m.data != nil {
		if err := unix.Munmap(m.data); err != nil {
			errs = append(errs, fmt.Errorf("munmap failed: %w", err))
		}
		m.data = nil
	}

	if m.file != nil {
		if err := m.file.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close failed: %w", err))
		}
		m.file = nil
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (m *MappedFile) Remove() error {
	path := m.path
	if err := m.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

// AsPtr returns an unsafe pointer to the mapped data.
func (m *MappedFile) AsPtr() unsafe.Pointer {
	return unsafe.Pointer(&m.data[0])
}

func (m *MappedFile) ReadVersion() uint64 {
	ptr := (*uint64)(unsafe.Pointer(&m.data[0]))
	return *ptr
}
