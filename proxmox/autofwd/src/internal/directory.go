package internal

import (
	"autofwd/src/logger"
	"bufio"
	"fmt"
	"os"
)

// Create a directory if the dir doesn't exist or if it's a file.
func CreateDirectory(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		if err = os.Mkdir(path, 0o644); err != nil {
			return fmt.Errorf("error failed to create %s: %s", path, err)
		}
	}
	if !s.IsDir() {
		if err = os.Mkdir(path, 0o644); err != nil {
			return fmt.Errorf("error failed to create %s: %s", path, err)
		}
	}
	return nil
}

// Return a *os.File that allow you to read and write, if allowed to write and read.
func OpenFile(path string) (*bufio.ReadWriter, error) {
	s, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error failed to open %s: %s", path, err)
	}
	if mod := s.Mode(); mod != 0 {
		if !mod.Perm().IsRegular() {
			return nil, fmt.Errorf("error file is not regular file")
		}

		logger.Printf("mod: %d, mod is writeable: %d\n", mod, mod&0200)
		if mod&0200 == 0 {
			return nil, fmt.Errorf("error file doesn't have user permision to read and write")
		}
	}
	o, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error failed to open %s: %s", path, err)
	}
	reader := bufio.NewReader(o)
	writer := bufio.NewWriter(o)
	if reader == nil || writer == nil {
		return nil, fmt.Errorf("error coudln't create a reader&writer")
	}
	if rw := bufio.NewReadWriter(reader, writer); rw != nil {
		return rw, nil
	}
	return nil, fmt.Errorf("error coudln't create a reader&writer")
}
