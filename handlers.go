package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/MohitPanchariya/goRed/resp"
)

// `redisValue` represents a key's value
type redisValue struct {
	expire    time.Time // expiration timestamp(unix milliseconds)
	valueType string    // type of value held by the key
	value     []byte    // the value stored as bytes
}

// store is a concurrent safe map
type store struct {
	lock sync.Mutex
	db   map[string]redisValue
}

// `newStore` returns an instance of `store`
func newStore() *store {
	s := store{
		db: make(map[string]redisValue),
	}
	return &s
}

// `get` is used to retrieve the value of a key
// in a concurrency safe manner
func (s *store) get(key string) ([]byte, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	value, ok := s.db[key]
	if !ok {
		return nil, ok
	}
	return value.value, ok
}

// `set` is used to set the value of a key in a
// concurrency safe manner
func (s *store) set(key string, value []byte, expire time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.db[key] = redisValue{
		valueType: "string",
		value:     value,
		expire:    expire,
	}
}

// PING command returns PONG
func ping(args [][]byte) ([]byte, error) {
	var response resp.SimpleString
	if len(args) == 0 {
		response.Data = "PONG"
	} else {
		response.Data = string(args[0])
	}
	serialisedData, serialisationError := response.Serialise()
	if serialisationError != nil {
		return nil, serialisationError
	}
	return serialisedData, serialisationError
}

// ECHO command echoes back the data to the client
func echo(args [][]byte) ([]byte, error) {
	if len(args) == 0 {
		return nil, resp.ErrInvalidClientData
	}
	var response resp.BulkString
	response.Data = args[0]
	response.Size = len(response.Data)
	serialisedData, serialisationError := response.Serialise()
	if serialisationError != nil {
		return nil, serialisationError
	}
	return serialisedData, nil
}

// GET command is used to retrieve the value of a key
func get(args [][]byte, s *store) ([]byte, error) {
	data, ok := s.get(string(args[0]))
	if !ok {
		var null resp.BulkString
		null.Size = -1
		return null.Serialise()
	}
	// get returns bulk strings
	var response resp.BulkString
	response.Data = data
	response.Size = len(data)
	return response.Serialise()
}

// SET command is used to set the value of a key
func set(args [][]byte, s *store) ([]byte, error) {
	var nx, xx, ex, px, exat, pxat, get bool
	setCounter := 0
	expiryOptionCounter := 0
	timeArgs := 0
	key := string(args[0])
	value := args[1]
	currentValue, keyExists := s.get(key)
	for i := 2; i < len(args); i++ {
		switch string(args[i]) {
		case "NX":
			setCounter++
			nx = true
		case "XX":
			setCounter++
			xx = true
		case "EX":
			expiryOptionCounter++
			timeArgs = i + 1
			ex = true
		case "PX":
			expiryOptionCounter++
			timeArgs = i + 1
			px = true
		case "EXAT":
			expiryOptionCounter++
			timeArgs = i + 1
			exat = true
		case "PXAT":
			expiryOptionCounter++
			timeArgs = i + 1
			pxat = true
		}
	}
	// error cases
	if (setCounter > 1) || (expiryOptionCounter > 1) || (timeArgs >= len(args)) {
		response := resp.SimpleError{
			Data: "invalid syntax",
		}
		serialised, err := response.Serialise()
		if err != nil {
			return nil, err
		}
		return serialised, nil
	}
	// cases where key shouldn't be set
	if (nx && keyExists) || (xx && !keyExists) {
		var response resp.RESPDatatype
		// if the key doesn't exist, nil must be returned in either
		// of the above cases
		if get && keyExists {
			data, _ := s.get(key)
			response = &resp.SimpleString{
				Data: string(data),
			}
		} else {
			response = &resp.BulkString{
				Size: -1,
			}
		}
		serialised, err := response.Serialise()
		if err != nil {
			return nil, err
		}
		return serialised, err
	}

	var expiration time.Time
	var parsedTime int
	var err error
	if expiryOptionCounter > 0 {
		parsedTime, err = strconv.Atoi(string(args[timeArgs]))
		if err != nil {
			return nil, err
		}
		currentTime := time.Now()
		if ex {
			expiration = currentTime.Add(time.Second * time.Duration(parsedTime))
		} else if px {
			expiration = currentTime.Add(time.Millisecond * time.Duration(parsedTime))
		} else if exat {
			expiration = time.Unix(int64(parsedTime), 0)
		} else if pxat {
			expiration = time.UnixMilli(int64(parsedTime))
		}
	}
	s.set(key, value, expiration)
	var response resp.RESPDatatype
	if get && keyExists {
		if keyExists {
			response = &resp.SimpleString{
				Data: string(currentValue),
			}
		} else {
			response = &resp.BulkString{
				Size: -1,
			}
		}
	} else {
		response = &resp.SimpleString{
			Data: "OK",
		}
	}
	serialised, err := response.Serialise()
	if err != nil {
		return nil, err
	}
	return serialised, nil
}
