package main

import (
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
func (s *store) set(key string, value []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.db[key] = redisValue{
		valueType: "string",
		value:     value,
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
	s.set(string(args[0]), args[1])
	var response resp.SimpleString
	response.Data = "OK"
	return response.Serialise()
}
