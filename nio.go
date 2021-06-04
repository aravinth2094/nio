package nio

import (
	"strings"
	"sync"
)

// ConvertToUpperCase chaining utility function for converting data to uppercase
var ConvertToUpperCase = func(data []byte, err error) ([]byte, bool, error) {
	if err != nil {
		return nil, true, err
	}
	return []byte(strings.ToUpper(string(data))), false, nil
}

// WaitForBytes chaining utility function for waiting for certain amount of bytes before passing the data further down the chain
var WaitForBytes = func(bytes int) func(data []byte, err error) ([]byte, bool, error) {
	return func(data []byte, err error) ([]byte, bool, error) {
		if len(data) < bytes {
			return nil, true, nil
		}
		return data, false, nil
	}
}

// ReaderWriterCloser defines signatures for Reader, Writer and Closer
type ReaderWriterCloser interface {
	Reader
	Writer
	Closer
}

// Reader defines signature for Read methods
type Reader interface {
	Read(*sync.WaitGroup, ...func([]byte, error) ([]byte, bool, error))
	RemoveReadCallback(int) error
}

// Writer defines signature for GetWriter method
type Writer interface {
	GetWriter(...func([]byte, error) ([]byte, bool, error)) func([]byte, func(int, error))
	Write([]byte, func(int, error))
}

// Closer defines signature for Close method
type Closer interface {
	Close() error
}
