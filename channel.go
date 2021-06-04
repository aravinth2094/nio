package nio

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// Channel holds the connection details
type Channel struct {
	protocol     string
	host         string
	port         int
	conn         net.Conn
	maxRead      int
	readBuffer   []byte
	writeBuffer  []byte
	writeChannel chan writeComplete
	readTimeout  time.Duration
	writeTimeout time.Duration
	readChain    []func(data []byte, err error) ([]byte, bool, error)
}

type writeComplete struct {
	data     []byte
	callback func(int, error)
}

var ErrEmptyCallbacks error = errors.New("cannot remove read callback: There must be atleast one callback in the chain")

// Read function starts an asynchrnous read routine that reads data from socket and chains the bytes through callbacks passed in the variadic argument
func (rw *Channel) Read(wg *sync.WaitGroup, callbacks ...func(data []byte, err error) ([]byte, bool, error)) {
	rw.readChain = callbacks
	go func() {
		defer wg.Done()
		rw.conn.SetReadDeadline(time.Now().Add(rw.readTimeout))
		for {
			drop := false
			incoming := make([]byte, rw.maxRead)
			n, err := rw.conn.Read(incoming)
			if err != nil {
				if netError, ok := err.(net.Error); ok && netError.Timeout() {
					rw.conn.SetReadDeadline(time.Now().Add(rw.readTimeout))
					continue
				} else if err == io.EOF {
					rw.Close()
					return
				}
			}
			rw.readBuffer = append(rw.readBuffer, incoming[:n]...)
			readData := rw.readBuffer
			for _, callback := range rw.readChain {
				if callback != nil {
					readData, drop, err = callback(readData, err)
				}
				if drop {
					break
				}
			}
			if !drop {
				rw.resetReadBuffer()
			}
		}
	}()
}

// RemoveReadCallback will remove the callback padding in Read function based on the index
func (rw *Channel) RemoveReadCallback(index int) error {
	if index < 0 || index >= len(rw.readChain) {
		return errors.New("cannot remove read callback: Invalid index " + strconv.Itoa(index))
	}
	if len(rw.readChain) == 1 {
		return ErrEmptyCallbacks
	}
	tempReadChain := make([]func(data []byte, err error) ([]byte, bool, error), len(rw.readChain)-2)
	tempReadChain = append(tempReadChain, rw.readChain[0:index]...)
	rw.readChain = append(tempReadChain, rw.readChain[index+1:]...)
	return nil
}

func (rw *Channel) writeRoutine() {
	rw.conn.SetWriteDeadline(time.Now().Add(rw.writeTimeout))
	for writeStructure := range rw.writeChannel {
		n, err := rw.conn.Write(writeStructure.data)
		if netError, ok := err.(net.Error); ok && netError.Timeout() {
			rw.conn.SetWriteDeadline(time.Now().Add(rw.writeTimeout))
			continue
		} else if err == io.EOF {
			rw.Close()
		}
		writeStructure.callback(n, err)
	}
}

// GetWriter returns a writer function into which data can be written and provided a callback for complete future
func (rw *Channel) GetWriter(callbacks ...func([]byte, error) ([]byte, bool, error)) func([]byte, func(int, error)) {
	return func(data []byte, writeCompleteCallback func(int, error)) {
		rw.writeBuffer = append(rw.writeBuffer, data...)
		var err error = nil
		drop := false
		defer func() {
			if !drop {
				rw.resetWriteBuffer()
			}
		}()
		for _, callback := range callbacks {
			rw.writeBuffer, drop, err = callback(rw.writeBuffer, err)
			if drop {
				break
			}
		}
		rw.writeChannel <- writeComplete{
			data:     rw.writeBuffer,
			callback: writeCompleteCallback,
		}
	}
}

func (rw *Channel) Write(data []byte, callback func(int, error)) {
	rw.GetWriter()(data, callback)
}

// Close the channels and connection
func (rw *Channel) Close() error {
	close(rw.writeChannel)
	return rw.conn.Close()
}

func (rw *Channel) resetReadBuffer() {
	rw.readBuffer = []byte{}
}

func (rw *Channel) resetWriteBuffer() {
	rw.writeBuffer = []byte{}
}

// GetChannel constructor for Channel structure
func GetChannel(protocol, host string, port int, secureConfig *tls.Config) (ReaderWriterCloser, error) {
	var conn net.Conn
	var err error
	conn, err = net.Dial(protocol, host+":"+strconv.Itoa(port))
	if err != nil {
		return nil, err
	}
	if protocol == "tcp" {
		conn.(*net.TCPConn).SetKeepAlive(true)
		conn.(*net.TCPConn).SetKeepAlivePeriod(30 * time.Second)
	}
	if secureConfig != nil {
		conn = tls.Client(conn, secureConfig)
	}
	var readerWriter ReaderWriterCloser = &Channel{
		protocol:     protocol,
		host:         host,
		port:         port,
		conn:         conn,
		maxRead:      8 * 1024,
		readBuffer:   make([]byte, 0),
		writeBuffer:  make([]byte, 0),
		writeChannel: make(chan writeComplete, 100),
		readTimeout:  60 * time.Second,
		writeTimeout: 60 * time.Second,
	}
	go readerWriter.(*Channel).writeRoutine()
	return readerWriter, nil
}
