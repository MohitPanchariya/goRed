package resp

import "errors"

var errInvalidDeserialiser = errors.New("data passed doesn't match deserialiser data typ")
var errTerminatorNotFound = errors.New("terminator not found")
