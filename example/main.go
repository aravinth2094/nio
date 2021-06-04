package main

import (
	"crypto/tls"
	"io"
	"log"
	"sync"

	nio "github.com/aravinth2094/nio"
)

func main() {

	printString := func(data []byte, err error) ([]byte, bool, error) {
		if err != nil {
			if err == io.EOF {
				log.Fatalln(err)
			}
			return nil, false, err
		}
		log.Println(string(data))
		return data, false, nil
	}

	printBytesWritten := func(bytesWritten int, err error) {
		if err != nil {
			if err == io.EOF {
				log.Fatalln(err)
			} else {
				log.Println(err)
			}
		} else {
			log.Println(bytesWritten, "bytes written")
		}
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel, err := nio.GetChannel("tcp", "localhost", 8080, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatalln(err)
	}
	write := channel.GetWriter(nio.ConvertToUpperCase, printString)
	channel.Read(wg, nio.ConvertToUpperCase, printString, func(data []byte, err error) ([]byte, bool, error) {
		if err != nil {
			return data, false, err
		}
		write(append([]byte("hello "), data...), printBytesWritten)
		return data, false, nil
	})
	channel.Write([]byte("Hi"), printBytesWritten)
	wg.Wait()
}
