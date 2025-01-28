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
	// TODO: dispatch to handlers
	reader := bufio.NewReader(c)
	// reuse same error object
	var e resp.SimpleError
	// client always sends an array of bulk strings
	data, err := reader.ReadBytes('\n')
	if err != nil {
		e.Data = resp.ErrTerminatorNotFound.Error()
		serialisedError, serialisationError := e.Serialise()
		if serialisationError != nil {
			c.Close()
		}
		c.Write(serialisedError)
		c.Close()
	}
	if string(data[0]) != resp.ARRAY_IDENTIFIER {
		e.Data = resp.ErrInvalidClientData.Error()
		serialisedError, serialisationError := e.Serialise()
		if serialisationError != nil {
			c.Close()
		}
		c.Write(serialisedError)
		c.Close()
	}
	// find the size of the array
	size, err := strconv.Atoi(string(data[1 : len(data)-resp.TERMINATOR_SIZE]))
	if err != nil {
		e.Data = resp.ErrLengthExtraction.Error()
		serialisedError, serialisationError := e.Serialise()
		if serialisationError != nil {
			c.Close()
		}
		c.Write(serialisedError)
		c.Close()
	}
	for i := 0; i < size; i++ {
		// read one bulk string at a time
		bulkString, err := reader.ReadBytes('\n')
		if err != nil {
			e.Data = resp.ErrTerminatorNotFound.Error()
			serialisedError, serialisationError := e.Serialise()
			if serialisationError != nil {
				c.Close()
			}
			c.Write(serialisedError)
			c.Close()
		}
		// extract length
		length, err := strconv.Atoi(string(data[1 : len(bulkString)-resp.TERMINATOR_SIZE]))
		if err != nil {
			e.Data = resp.ErrLengthExtraction.Error()
			serialisedError, serialisationError := e.Serialise()
			if serialisationError != nil {
				c.Close()
			}
			c.Write(serialisedError)
			c.Close()
		}
		bulkStrinData := make([]byte, length)
		copied, err := io.ReadFull(reader, bulkStrinData)
		if err != nil {
			e.Data = err.Error()
			serialisedError, serialisationError := e.Serialise()
			if serialisationError != nil {
				c.Close()
			}
			c.Write(serialisedError)
			c.Close()
		}
		if copied != len(bulkStrinData) {
			e.Data = resp.ErrBulkStringDataSize.Error()
			serialisedError, serialisationError := e.Serialise()
			if serialisationError != nil {
				c.Close()
			}
			c.Write(serialisedError)
			c.Close()
		}
		// read the terminator
		terminator, err := reader.ReadBytes('\n')
		if err != nil {
			e.Data = resp.ErrTerminatorNotFound.Error()
			serialisedError, serialisationError := e.Serialise()
			if serialisationError != nil {
				c.Close()
			}
			c.Write(serialisedError)
			c.Close()
		}
		bulkString = append(bulkString, bulkStrinData...)
		bulkString = append(bulkString, terminator...)
		// add bulk string to the data
		data = append(data, bulkString...)
	}
	c.Write(data)
	c.Close()
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
