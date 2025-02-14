package resp

import "errors"

var (
	ErrInvalidDeserialiser    = errors.New("data passed doesn't match deserialiser data type")
	ErrTerminatorNotFound     = errors.New("terminator not found")
	ErrIntegerConversion      = errors.New("failed to extract integer")
	ErrLengthExtraction       = errors.New("failed to extract length")
	ErrUnidentifiedType       = errors.New("unidentified data type")
	ErrInvalidClientData      = errors.New("client messages not an array of bulk strings")
	ErrBulkStringDataSize     = errors.New("bulk string size and data don't match")
	ErrFailedToCreateDumpFile = errors.New("failed to make the dump file")
	ErrFailedToDumpDB         = errors.New("failed to write to the database file")
)
