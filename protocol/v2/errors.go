package protocol

import (
	"errors"
)

// The errors returned by the device if the processing of the instruction fails.
// See https://emanual.robotis.com/docs/en/dxl/protocol2/#error for more details.
var (
	ErrDeviceError      = errors.New("processing error - device error")
	ErrResultError      = errors.New("processing error - result failed")
	ErrInstructionError = errors.New("processing error - instruction error")
	ErrDeviceCRCError   = errors.New("processing error - crc verification  error")
	ErrDataRangeError   = errors.New("processing error - data range error")
	ErrDataLengthError  = errors.New("processing error - data length error")
	ErrDataLimitError   = errors.New("processing error - data limit error")
	ErrAccessError      = errors.New("processing error - access error")
)

var (
	ErrInvalidID           = errors.New("invalid device ID")
	ErrReadTimeout         = errors.New("read wait timeout")
	ErrTruncatedStatus     = errors.New("status packet truncated")
	ErrMalformedStatus     = errors.New("malformed status packet")
	ErrInvalidStatusLength = errors.New("invalid status packet length value")
	ErrStatusCRCInvalid    = errors.New("status packet crc check failed")
)

var (
	ErrUnexpectedParamCount = errors.New("unexpected parameter count")
	ErrNoStatusOnBroadcast  = errors.New("instruction does not respond to Broadcast ID")
	ErrMinOneIDRequired     = errors.New("at least one ID is required")
)
