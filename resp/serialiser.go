package resp

type RESPDatatype interface {
	// serialise the data into a byte stream
	serialise() ([]byte, error)
	// deserialise the byte stream into a RESPDatatype
	// returns the last read index in the byte slice
	deserialise([]byte) (int, error)
}
