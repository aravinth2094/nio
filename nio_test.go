package nio

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func startServer(port int, t *testing.T) {
	server, err := net.Listen("tcp4", "localhost:"+strconv.Itoa(port))
	if err != nil {
		t.Error(err)
	}
	defer server.Close()
	client, err := server.Accept()
	if err != nil {
		t.Error(err)
	}
	defer client.Close()
	data := make([]byte, 1024)
	n, err := client.Read(data)
	if err != nil {
		t.Error(err)
	}
	client.Write(append([]byte("Hello "), data[:n]...))
}

func TestResponse(t *testing.T) {
	go startServer(8080, t)
	channel, err := GetChannel("tcp", "localhost", 8080, nil)
	if err != nil {
		t.Error(err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel.Write([]byte("World"), func(bytesWritten int, err error) {
		if err != nil {
			t.Error(err)
		}
	})
	channel.Read(wg, ConvertToUpperCase, func(data []byte, err error) ([]byte, bool, error) {
		if err != nil {
			t.Error(err)
			return nil, false, err
		}
		if strings.Compare(string(data), "HELLO WORLD") != 0 {
			t.Error("Not equal", string(data))
		}
		return data, false, nil
	})
	wg.Wait()
}

func TestRemoveCallback(t *testing.T) {
	go startServer(8080, t)
	channel, err := GetChannel("tcp", "localhost", 8080, nil)
	if err != nil {
		t.Error(err)
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel.Read(wg, ConvertToUpperCase, func(data []byte, err error) ([]byte, bool, error) {
		if err != nil {
			t.Error(err)
			return nil, false, err
		}
		if strings.Compare(string(data), "Hello World") != 0 {
			t.Error("Not equal", string(data))
		}
		return data, false, nil
	})
	err = channel.RemoveReadCallback(0)
	if err != nil {
		t.Error(err)
	}
	channel.Write([]byte("World"), func(bytesWritten int, err error) {
		if err != nil {
			t.Error(err)
		}
	})
	err = channel.RemoveReadCallback(0)
	if err != ErrEmptyCallbacks {
		t.Error(err)
	}
	wg.Wait()
}
