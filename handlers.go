package main

import (
	"github.com/MohitPanchariya/goRed/resp"
)

// PING command returns PONG
func ping(args [][]byte) []byte {
	var response resp.SimpleString
	if len(args) == 0 {
		response.Data = "PONG"
	} else {
		response.Data = string(args[0])
	}
	serialiseData, serialisationError := response.Serialise()
	if serialisationError != nil {
		return nil
	}
	return serialiseData
}
