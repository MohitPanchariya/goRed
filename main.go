package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"

	"github.com/MohitPanchariya/goRed/resp"
)

var keyValueStore = newStore()

func dispatch(c net.Conn) {
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
	reader := bufio.NewReader(c)
	// read from the TCP connection until its closed
	for {
		// command stores the command and arguments passed
		var command [][]byte
		// arrayToken is made up of ARRAY_IDENTIFIER<size>TERMINATOR
		arrayToken, err := reader.ReadBytes('\n')
		if err != nil {
			// no data read and EOF implies that the client closed the connection
			if len(arrayToken) == 0 {
				return nil
			} else { // abrupt termination of the connection
				return resp.ErrTerminatorNotFound
			}
		}
		if string(arrayToken[0]) != resp.ARRAY_IDENTIFIER {
			return resp.ErrInvalidClientData
		}
		arraySize, err := strconv.Atoi(string(arrayToken[1 : len(arrayToken)-resp.TERMINATOR_SIZE]))
		if err != nil {
			return resp.ErrLengthExtraction
		}
		// read one bulk string at a time
		for i := 0; i < arraySize; i++ {
			bulkString, err := reader.ReadBytes('\n')
			if err != nil {
				return resp.ErrTerminatorNotFound
			}
			// extract length
			length, err := strconv.Atoi(string(bulkString[1 : len(bulkString)-resp.TERMINATOR_SIZE]))
			if err != nil {
				return resp.ErrLengthExtraction
			}
			bulkStringData := make([]byte, length)
			copied, err := io.ReadFull(reader, bulkStringData)
			if err != nil {
				return err
			}
			if copied != len(bulkStringData) {
				return resp.ErrBulkStringDataSize
			}
			// read the terminator
			_, err = reader.ReadBytes('\n')
			if err != nil {
				return resp.ErrTerminatorNotFound
			}
			// add command/arg to the data without the TERMINATOR
			command = append(command, bulkStringData)
		}
		var serialisedData []byte
		switch string(command[0]) {
		case "PING":
			serialisedData, err = ping(command[1:])
		case "ECHO":
			serialisedData, err = echo(command[1:])
		case "GET":
			serialisedData, err = get(command[1:], keyValueStore)
		case "SET":
			serialisedData, err = set(command[1:], keyValueStore)
		case "EXISTS":
			serialisedData, err = exists(command[1:], keyValueStore)
		case "DEL":
			serialisedData, err = del(command[1:], keyValueStore)
		case "INCR":
			serialisedData, err = incr(command[1:], keyValueStore)
		case "DECR":
			serialisedData, err = decr(command[1:], keyValueStore)
		case "LPUSH":
			serialisedData, err = lpush(command[1:], keyValueStore)
		case "RPUSH":
			serialisedData, err = rpush(command[1:], keyValueStore)
		case "LRANGE":
			serialisedData, err = lrange(command[1:], keyValueStore)
		}
		if err != nil {
			return err
		} else {
			c.Write(serialisedData)
		}
	}
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
		go dispatch(conn)
	}
}
