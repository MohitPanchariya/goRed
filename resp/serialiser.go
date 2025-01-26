package resp

const (
	SIMPLE_STRING_IDENTIFIER = "+"
	SIMPLE_ERROR_IDENTIFIER  = "-"
	INTEGER_IDENTIFIER       = ":"
	BULK_STRING_IDENTIFIER   = "$"
	ARRAY_IDENTIFIER         = "*"
	TERMINATOR               = "\r\n"
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

// Serialise is used to serialise a SimpleString into
// the RESP format
func (s *SimpleString) Serialise() ([]byte, error) {
	return []byte(SIMPLE_STRING_IDENTIFIER + s.Data + TERMINATOR), nil
}

// Deserialise is used to convert data into a SimpleString
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
