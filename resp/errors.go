package resp

import "errors"

var ErrInvalidDeserialiser = errors.New("data passed doesn't match deserialiser data type")
var ErrTerminatorNotFound = errors.New("terminator not found")
var ErrIntegerConversion = errors.New("failed to extract integer")
var ErrLengthExtraction = errors.New("failed to extract length")
var ErrUnidentifiedType = errors.New("unidentified data type")
var ErrInvalidClientData = errors.New("client messages not an array of bulk strings")
var ErrBulkStringDataSize = errors.New("bulk string size and data don't match")
