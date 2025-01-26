package resp

import "errors"

var errInvalidDeserialiser = errors.New("data passed doesn't match deserialiser data type")
var errTerminatorNotFound = errors.New("terminator not found")
var errIntegerConversion = errors.New("failed to extract integer")
var errLengthExtraction = errors.New("failed to extract length")
var errUnidentifiedType = errors.New("unidentified data type")
