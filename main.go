package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/MohitPanchariya/goRed/resp"
)

func dispatcher(c net.Conn) {
	var respError resp.SimpleError
	err := dispatchHelper(c)
	if err != nil {
		respError.Data = err.Error()
		serialisedError, serialisationError := respError.Serialise()
		if serialisationError != nil {
			c.Close()
		}
		c.Write(serialisedError)
		c.Close()
	}
}

func dispatchHelper(c net.Conn) error {
	// TODO: dispatch to handlers
	reader := bufio.NewReader(c)
	// client always sends an array of bulk strings
	data, err := reader.ReadBytes('\n')
	if err != nil {
		return resp.ErrTerminatorNotFound
	}
	if string(data[0]) != resp.ARRAY_IDENTIFIER {
		return resp.ErrInvalidClientData
	}
	// find the size of the array
	size, err := strconv.Atoi(string(data[1 : len(data)-resp.TERMINATOR_SIZE]))
	if err != nil {
		return resp.ErrLengthExtraction
	}
	for i := 0; i < size; i++ {
		// read one bulk string at a time
		bulkString, err := reader.ReadBytes('\n')
		if err != nil {
			return resp.ErrTerminatorNotFound
		}
		// extract length
		length, err := strconv.Atoi(string(data[1 : len(bulkString)-resp.TERMINATOR_SIZE]))
		if err != nil {
			return resp.ErrLengthExtraction
		}
		bulkStrinData := make([]byte, length)
		copied, err := io.ReadFull(reader, bulkStrinData)
		if err != nil {
			return err
		}
		if copied != len(bulkStrinData) {
			return resp.ErrBulkStringDataSize
		}
		// read the terminator
		terminator, err := reader.ReadBytes('\n')
		if err != nil {
			return resp.ErrTerminatorNotFound
		}
		bulkString = append(bulkString, bulkStrinData...)
		bulkString = append(bulkString, terminator...)
		// add bulk string to the data
		data = append(data, bulkString...)
	}
	c.Write(data)
	c.Close()
	return nil
}

func main() {
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go dispatcher(conn)
	}
}
