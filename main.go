package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

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
		case "SAVE":
			serialisedData, err = save(command[1:], keyValueStore)
		default:
			return resp.ErrInvalidCommand
		}
		if err != nil {
			return err
		} else {
			c.Write(serialisedData)
		}
	}
}

// `extractKeyValuePair` extracts a key value pair
func extractKeyValuePair(reader *bufio.Reader) (string, redisValue, error) {
	var value redisValue
	keyToken, err := reader.ReadBytes('\n')
	if err != nil {
		// no bytes read implies the reader has reach EOF
		if len(keyToken) == 0 {
			return "", value, io.EOF
		}
		return "", value, err
	}
	key := string(keyToken[1 : len(keyToken)-resp.TERMINATOR_SIZE])
	expireTimeToken, err := reader.ReadBytes('\n')
	if err != nil {
		return "", value, err
	}
	expireTime, err := time.Parse(time.UnixDate, string(expireTimeToken[1:len(expireTimeToken)-resp.TERMINATOR_SIZE]))
	if err != nil {
		return "", value, err
	}
	valueTypeToken, err := reader.ReadBytes('\n')
	if err != nil {
		return "", value, err
	}
	valueType := string(valueTypeToken[1 : len(valueTypeToken)-resp.TERMINATOR_SIZE])
	if valueType == "string" {
		bulkString, err := reader.ReadBytes('\n')
		if err != nil {
			return "", value, err
		}
		length, err := strconv.Atoi(string(bulkString[1 : len(bulkString)-resp.TERMINATOR_SIZE]))
		if err != nil {
			return "", value, err
		}
		bulkStringData := make([]byte, length)
		copied, err := io.ReadFull(reader, bulkStringData)
		if err != nil {
			return "", value, err
		}
		if copied != len(bulkStringData) {
			return "", value, resp.ErrBulkStringDataSize
		}
		// read the terminator
		_, err = reader.ReadBytes('\n')
		if err != nil {
			return "", value, resp.ErrTerminatorNotFound
		}
		value.value = bulkStringData
		value.expire = expireTime
		value.valueType = "string"
	} else { // list is stored as an array of bulk strings
		// arrayToken is made up of ARRAY_IDENTIFIER<size>TERMINATOR
		arrayToken, err := reader.ReadBytes('\n')
		if err != nil {
			return "", value, resp.ErrTerminatorNotFound
		}
		if string(arrayToken[0]) != resp.ARRAY_IDENTIFIER {
			return "", value, resp.ErrInvalidClientData
		}
		arraySize, err := strconv.Atoi(string(arrayToken[1 : len(arrayToken)-resp.TERMINATOR_SIZE]))
		if err != nil {
			return "", value, resp.ErrLengthExtraction
		}
		nodes := make([]*node, arraySize)
		// read one bulk string at a time
		for i := 0; i < arraySize; i++ {
			bulkString, err := reader.ReadBytes('\n')
			if err != nil {
				return "", value, resp.ErrTerminatorNotFound
			}
			// extract length
			length, err := strconv.Atoi(string(bulkString[1 : len(bulkString)-resp.TERMINATOR_SIZE]))
			if err != nil {
				return "", value, resp.ErrLengthExtraction
			}
			bulkStringData := make([]byte, length)
			copied, err := io.ReadFull(reader, bulkStringData)
			if err != nil {
				return "", value, err
			}
			if copied != len(bulkStringData) {
				return "", value, resp.ErrBulkStringDataSize
			}
			nodes[i] = &node{
				data: bulkStringData,
			}
			// read the terminator
			_, err = reader.ReadBytes('\n')
			if err != nil {
				return "", value, resp.ErrTerminatorNotFound
			}
		}
		// add the values to a list
		list := newList()
		value.value = list
		list.tpush(nodes)
	}
	return key, value, nil
}

func loadFromDB(file *os.File) error {
	reader := bufio.NewReader(file)
	for {
		key, value, err := extractKeyValuePair(reader)
		if err != nil {
			// finished reading the dump
			if errors.Is(err, io.EOF) {
				break
			}
		}
		// store the key value pair in the database
		keyValueStore.db[key] = value
	}
	return nil
}

func main() {
	if len(os.Args) > 1 {
		dbFilePath := os.Args[1]
		dbFile, err := os.Open(dbFilePath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = loadFromDB(dbFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
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
