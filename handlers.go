package main

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/MohitPanchariya/goRed/resp"
)

// `redisValue` represents a key's value
type redisValue struct {
	expire    time.Time // expiration timestamp(unix milliseconds)
	valueType string    // type of value held by the key
	value     any
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
func (s *store) get(key string) (*redisValue, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	value, ok := s.db[key]
	if !ok {
		return nil, ok
	}
	if !value.expire.IsZero() {
		expired := value.expire.Compare(time.Now())
		if expired == -1 {
			// delete the key - This is a passive delete strategy
			delete(s.db, key)
			return nil, false
		}
	}
	return &value, ok
}

// `set` is used to set the value of a key in a
// concurrency safe manner
func (s *store) set(key string, value *redisValue) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.db[key] = *value
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
	value, ok := s.get(string(args[0]))
	if !ok {
		var null resp.BulkString
		null.Size = -1
		return null.Serialise()
	}
	// check if value is of type string
	if value.valueType != "string" {
		response := resp.SimpleError{
			Data: "value is not of string type",
		}
		return response.Serialise()
	}
	// get returns bulk strings
	var response resp.BulkString
	response.Data = value.value.([]byte)
	response.Size = len(response.Data)
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
			value, _ := s.get(key)
			response = &resp.SimpleString{
				Data: string(value.value.([]byte)),
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
	if currentValue == nil {
		currentValue = &redisValue{}
	}
	currentValue.value = value
	currentValue.expire = expiration
	currentValue.valueType = "string"
	s.set(key, currentValue)
	var response resp.RESPDatatype
	if get && keyExists {
		if keyExists {
			response = &resp.SimpleString{
				Data: string(currentValue.value.([]byte)),
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

// EXISTS command checks if a key(s) exists
func exists(args [][]byte, s *store) ([]byte, error) {
	existsCounter := 0
	for i := 0; i < len(args); i++ {
		_, ok := s.get(string(args[i]))
		if ok {
			existsCounter++
		}
	}
	reply := resp.Integer{
		Data: int64(existsCounter),
	}
	serialisedData, err := reply.Serialise()
	if err != nil {
		return nil, err
	}
	return serialisedData, nil
}

// DEL command deletes a key(s)
func del(args [][]byte, s *store) ([]byte, error) {
	deleteCounter := 0
	for i := 0; i < len(args); i++ {
		_, ok := s.get(string(args[i]))
		if ok {
			deleteCounter++
			s.lock.Lock()
			delete(s.db, string(args[i]))
			s.lock.Unlock()
		}
	}
	resp := resp.Integer{
		Data: int64(deleteCounter),
	}
	serialised, err := resp.Serialise()
	if err != nil {
		return nil, err
	}
	return serialised, nil
}

// INCR command increments the number stored at key by one
func incr(args [][]byte, s *store) ([]byte, error) {
	key := string(args[0])
	response := resp.Integer{}
	value, ok := s.get(key)
	if !ok {
		value = &redisValue{
			value:     []byte("1"),
			valueType: "string",
		}
		s.set(key, value)
		response.Data = 1
	} else {
		if value.valueType != "string" {
			response := resp.SimpleError{
				Data: "value is not of numeric type",
			}
			return response.Serialise()
		}
		integer, err := strconv.Atoi(string(value.value.([]byte)))
		if err != nil {
			return nil, err
		}
		value.value = []byte(strconv.Itoa(integer + 1))
		s.set(key, value)
		response.Data = int64(integer + 1)
	}
	serialised, err := response.Serialise()
	return serialised, err
}

// DECR command decrements the number stored at key by one
func decr(args [][]byte, s *store) ([]byte, error) {
	key := string(args[0])
	response := resp.Integer{}
	value, ok := s.get(key)
	if !ok {
		value = &redisValue{
			value:     []byte("-1"),
			valueType: "string",
		}
		s.set(key, value)
		response.Data = -1
	} else {
		if value.valueType != "string" {
			response := resp.SimpleError{
				Data: "value is not of numeric type",
			}
			return response.Serialise()
		}
		integer, err := strconv.Atoi(string(value.value.([]byte)))
		if err != nil {
			return nil, err
		}
		value.value = []byte(strconv.Itoa(integer - 1))
		s.set(key, value)
		response.Data = int64(integer - 1)
	}
	serialised, err := response.Serialise()
	return serialised, err
}

// LPUSH command inserts value at the head of a list
func lpush(args [][]byte, s *store) ([]byte, error) {
	key := string(args[0])
	nodes := make([]*node, 0)
	for i := 1; i < len(args); i++ {
		nodes = append(nodes, &node{
			data: args[i],
		})
	}
	value, ok := s.get(key)
	var l *list
	if !ok {
		l = newList()
		value = &redisValue{
			value:     l,
			valueType: "list",
		}
		s.set(key, value)
	} else {
		l = value.value.(*list)
	}
	l.hpush(nodes)
	response := resp.Integer{
		Data: int64(l.length),
	}
	return response.Serialise()
}

// RPUSH command inserts value at the tail of a list
func rpush(args [][]byte, s *store) ([]byte, error) {
	key := string(args[0])
	nodes := make([]*node, 0)
	for i := 1; i < len(args); i++ {
		nodes = append(nodes, &node{
			data: args[i],
		})
	}
	value, ok := s.get(key)
	var l *list
	if !ok {
		l = newList()
		value = &redisValue{
			value:     l,
			valueType: "list",
		}
		s.set(key, value)
	} else {
		l = value.value.(*list)
	}
	l.tpush(nodes)
	response := resp.Integer{
		Data: int64(l.length),
	}
	return response.Serialise()
}

// convert a list to a RESP array of bulk strings
func (l *list) toRESPArray(start, end int) ([]byte, error) {
	var response resp.Array
	listPointer := l.head
	for i := 0; i < l.length && i < start; i++ {
		listPointer = listPointer.next
	}
	for i := start; i < l.length && i <= end; i++ {
		elem := resp.BulkString{
			Data: listPointer.data,
			Size: len(listPointer.data),
		}
		response.Elements = append(response.Elements, &elem)
		listPointer = listPointer.next
	}
	response.Size = len(response.Elements)
	return response.Serialise()
}

// LRANGE command returns specified elements of the list
// stored at key
func lrange(args [][]byte, s *store) ([]byte, error) {
	if len(args) < 3 {
		return nil, resp.ErrInvalidClientData
	}
	var response resp.Array
	response.Size = 0
	key := string(args[0])
	l, ok := s.get(key)
	if !ok {
		return response.Serialise()
	}
	start, err := strconv.Atoi(string(args[1]))
	if err != nil {
		return nil, resp.ErrIntegerConversion
	}
	end, err := strconv.Atoi(string(args[2]))
	if err != nil {
		return nil, resp.ErrInvalidClientData
	}
	if start > l.value.(*list).length {
		return response.Serialise()
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	return l.value.(*list).toRESPArray(start, end)
}

// SAVE command is used to save the database to disk
func save(args [][]byte, s *store) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// serialise the database
	db, err := os.Create("db.dump")
	if err != nil {
		return nil, resp.ErrFailedToCreateDumpFile
	}
	writer := bufio.NewWriter(db)
	for key, value := range s.db {
		keyBulk := resp.SimpleString{
			Data: string([]byte(key)),
		}
		serialisedKey, err := keyBulk.Serialise()
		if err != nil {
			return nil, err
		}
		expire := value.expire.Format(time.UnixDate)
		timeBulk := resp.SimpleString{
			Data: string([]byte(expire)),
		}
		serialisedExpire, err := timeBulk.Serialise()
		if err != nil {
			return nil, err
		}
		valueTypeBulk := resp.SimpleString{
			Data: string([]byte(value.valueType)),
		}
		serialisedValueType, err := valueTypeBulk.Serialise()
		if err != nil {
			return nil, err
		}
		var serialisedValue []byte
		// a list will be stored as an array of bulk string
		if value.valueType == "list" {
			serialisedValue, err = value.value.(*list).toRESPArray(0, value.value.(*list).length)
			if err != nil {
				return nil, err
			}
		} else {
			dataBulk := resp.BulkString{
				Data: value.value.([]byte),
				Size: len(value.value.([]byte)),
			}
			serialisedValue, err = dataBulk.Serialise()
			if err != nil {
				return nil, err
			}
		}
		serialisedData := bytes.Join([][]byte{serialisedKey, serialisedExpire, serialisedValueType, serialisedValue}, []byte(""))
		written, err := writer.Write(serialisedData)
		if err != nil || written < len(serialisedData) {
			return nil, resp.ErrFailedToDumpDB
		}
	}
	err = writer.Flush()
	if err != nil {
		return nil, resp.ErrFailedToDumpDB
	}
	response := resp.SimpleString{
		Data: "OK",
	}
	return response.Serialise()
}
