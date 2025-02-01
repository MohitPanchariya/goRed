package main

import (
	"github.com/MohitPanchariya/goRed/resp"
)

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
