package resp

import (
	"bytes"
	"strconv"
)

const (
	SIMPLE_STRING_IDENTIFIER = "+"
	SIMPLE_ERROR_IDENTIFIER  = "-"
	INTEGER_IDENTIFIER       = ":"
	BULK_STRING_IDENTIFIER   = "$"
	ARRAY_IDENTIFIER         = "*"
	TERMINATOR               = "\r\n"
	TERMINATOR_SIZE          = 2
)

// RESPDatatype is implemented by a redis
// data type
type RESPDatatype interface {
	// Serialise the data into a byte stream
	Serialise() ([]byte, error)
	// Deserialise the byte stream into a RESPDatatype
	// returns the last read index in the byte slice
	Deserialise([]byte) (int, error)
}

// tokenize extracts a token from the slice.
// It returns the index of the last byte read, a byte
// slice excluding the first byte(the identifier)
// and including all the bytes upto the first occurence
// of the terminator, \r\n. It returns an error, if the
// terminator is not found
func tokenize(data []byte) (int, []byte, error) {
	// ignore first byte and search for the terminator
	for i := 1; i < len(data); i++ {
		if data[i] == '\n' && data[i-1] == '\r' {
			token := make([]byte, (i-1)-1)
			copy(token, data[1:(i-1)])
			return i, token, nil
		}
	}
	return len(data) - 1, nil, errTerminatorNotFound
}

// SimpleString is a redis data type that implements
// the RESPDatatype interface
type SimpleString struct {
	Data string
}

// Serialise serialises a SimpleString into
// the RESP format
func (s *SimpleString) Serialise() ([]byte, error) {
	return []byte(SIMPLE_STRING_IDENTIFIER + s.Data + TERMINATOR), nil
}

// Deserialise converts data into a SimpleString
func (s *SimpleString) Deserialise(data []byte) (int, error) {
	// check data type
	if string(data[0]) != SIMPLE_STRING_IDENTIFIER {
		return 0, errInvalidDeserialiser
	}
	position, data, err := tokenize(data)
	if err != nil {
		return position, err
	}
	s.Data = string(data)
	return position, nil
}

// SimpleError is a redis data type that implements the
// RESPDatatype interface
type SimpleError struct {
	Data string
}

// Serialise serialises a SimpleError into the RESP
// format
func (s *SimpleError) Serialise() ([]byte, error) {
	return []byte(SIMPLE_ERROR_IDENTIFIER + s.Data + TERMINATOR), nil
}

// Deserialise converts data into a SimpleError
func (s *SimpleError) Deserialise(data []byte) (int, error) {
	// check if data is of type simple error
	if string(data[0]) != SIMPLE_ERROR_IDENTIFIER {
		return 0, errInvalidDeserialiser
	}
	position, token, err := tokenize(data)
	if err != nil {
		return position, err
	}
	s.Data = string(token)
	return position, nil
}

// Integer is a redis data type that implements the
// RESPDatatype interface
type Integer struct {
	Data int64
}

// Serialise serialises an Integer into the RESP format
func (i *Integer) Serialise() ([]byte, error) {
	return []byte(INTEGER_IDENTIFIER + strconv.Itoa(int(i.Data)) + TERMINATOR), nil
}

// Deserialise converts data into an Integer
func (i *Integer) Deserialise(data []byte) (int, error) {
	// check if data is of type Integer
	if string(data[0]) != INTEGER_IDENTIFIER {
		return 0, nil
	}
	position, data, err := tokenize(data)
	if err != nil {
		return position, err
	}
	num, err := strconv.Atoi(string(data))
	if err != nil {
		return position, errIntegerConversion
	}
	i.Data = int64(num)
	return position, nil
}

// BulkString is a redis data type that implement the
// RESPDatatype interface
type BulkString struct {
	Size int
	Data []byte
}

// Serialise serialises a BulkString into the RESP format
func (bs *BulkString) Serialise() ([]byte, error) {
	// null bulk string
	if bs.Size == -1 {
		return []byte(BULK_STRING_IDENTIFIER + "-1" + TERMINATOR), nil
	}
	// empty bulk string
	if bs.Size == 0 {
		return []byte(BULK_STRING_IDENTIFIER + "0" + TERMINATOR + TERMINATOR), nil
	}
	return []byte(BULK_STRING_IDENTIFIER + strconv.Itoa(bs.Size) + TERMINATOR + string(bs.Data) + TERMINATOR), nil
}

// Deserialise converts data into a BulkString
func (bs *BulkString) Deserialise(data []byte) (int, error) {
	// check if data is of type bulk string
	if string(data[0]) != BULK_STRING_IDENTIFIER {
		return 0, errInvalidDeserialiser
	}
	position, token, err := tokenize(data)
	if err != nil {
		return position, err
	}
	length, err := strconv.Atoi(string(token))
	if err != nil {
		return position, errLengthExtraction
	}
	bs.Size = length
	// null bulk string
	if bs.Size == -1 {
		bs.Data = nil
		return position, nil
	}
	if bs.Size == 0 {
		bs.Data = make([]byte, 0)
		return position + TERMINATOR_SIZE, nil
	}
	bs.Data = data[position+1 : position+1+length]
	return position + length + TERMINATOR_SIZE, nil
}

// Array is a redis data type that implements the
// RESPDatatype interface
type Array struct {
	Size     int
	Elements []RESPDatatype
}

// Serialise serialises an Array into the RESP format
func (a *Array) Serialise() ([]byte, error) {
	// null array
	if a.Size == -1 {
		return []byte(ARRAY_IDENTIFIER + "-1" + TERMINATOR), nil
	}
	// empty array
	if a.Size == 0 {
		return []byte(ARRAY_IDENTIFIER + "0" + TERMINATOR), nil
	}

	var serialised bytes.Buffer
	serialised.WriteString("*" + strconv.Itoa(a.Size) + TERMINATOR)
	for i := 0; i < len(a.Elements); i++ {
		serialisedElement, err := a.Elements[i].Serialise()
		if err != nil {
			return nil, err
		}
		serialised.Write(serialisedElement)
	}
	return serialised.Bytes(), nil
}

// Deserialise converts data into an Array
func (a *Array) Deserialise(data []byte) (int, error) {
	// check if data is of type array
	if string(data[0]) != ARRAY_IDENTIFIER {
		return 0, errInvalidDeserialiser
	}
	// null array
	if string(data[1]) == "-1" {
		a.Size = -1
		a.Elements = nil
		return 1, nil
	}
	// empty array
	if string(data[1]) == "0" {
		a.Size = 0
		a.Elements = make([]RESPDatatype, 0)
		return 1, nil
	}

	position, token, err := tokenize(data)
	if err != nil {
		return position, err
	}
	length, err := strconv.Atoi(string(token))
	if err != nil {
		return position, errLengthExtraction
	}
	a.Size = length
	a.Elements = make([]RESPDatatype, a.Size)

	for i := 0; i < a.Size; i++ {
		position++
		switch string(data[position]) {
		case SIMPLE_STRING_IDENTIFIER:
			a.Elements[i] = new(SimpleString)
			relativePos, err := a.Elements[i].Deserialise(data[position:])
			if err != nil {
				return position + relativePos, err
			}
			position += relativePos
		case SIMPLE_ERROR_IDENTIFIER:
			a.Elements[i] = new(SimpleError)
			relativePos, err := a.Elements[i].Deserialise(data[position:])
			if err != nil {
				return position + relativePos, err
			}
			position += relativePos
		case INTEGER_IDENTIFIER:
			a.Elements[i] = new(Integer)
			relativePos, err := a.Elements[i].Deserialise(data[position:])
			if err != nil {
				return position + relativePos, err
			}
			position += relativePos
		case BULK_STRING_IDENTIFIER:
			a.Elements[i] = new(BulkString)
			relativePos, err := a.Elements[i].Deserialise(data[position:])
			if err != nil {
				return position + relativePos, err
			}
			position += relativePos
		case ARRAY_IDENTIFIER:
			a.Elements[i] = new(Array)
			relativePos, err := a.Elements[i].Deserialise(data[position:])
			if err != nil {
				return position + relativePos, err
			}
			position += relativePos
		default:
			return position, errUnidentifiedType
		}
	}
	return position, nil
}
