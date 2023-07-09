package protocol

import (
	"errors"
)

var (
	errDeviceError      = errors.New("processing error - device error")
	errResultError      = errors.New("processing error - result failed")
	errInstructionError = errors.New("processing error - instruction error")
	errDeviceCRCError   = errors.New("processing error - crc verification  error")
	errDataRangeError   = errors.New("processing error - data range error")
	errDataLengthError  = errors.New("processing error - data length error")
	errDataLimitError   = errors.New("processing error - data limit error")
	errAccessError      = errors.New("processing error - access error")
)

var (
	errInvalidID           = errors.New("invalid device ID")
	errTruncatedStatus     = errors.New("status packet truncated")
	errMalformedStatus     = errors.New("malformed status packet")
	errInvalidStatusLength = errors.New("invalid status packet length value")
	errStatusCRCInvalid    = errors.New("status packet crc check failed")
)

var (
	errUnexpectedParamCount = errors.New("unexpected parameter count")
	errNoStatusOnBroadcast  = errors.New("instruction does not respond to Broadcast ID")
)
