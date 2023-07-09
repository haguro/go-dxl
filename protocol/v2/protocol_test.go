package protocol

import (
	"errors"
	"reflect"
	"testing"
)

func TestInstructionPacketBytes(t *testing.T) {
	testCases := []struct {
		name     string
		inst     *instruction
		expErr   error
		expBytes []byte
	}{
		{
			name: "Invalid ID",
			inst: &instruction{id: 0xFF,
				command: ping},
			expErr: errInvalidID,
		},
		{
			name: "Valid instruction with no params",
			inst: &instruction{
				id:      0x01,
				command: ping,
			},
			expBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x03, 0x00, 0x01, 0x19, 0x4E},
		},
		{
			name: "Valid instruction with single param",
			inst: &instruction{
				id:      0x01,
				command: reset,
				params:  []byte{0x01},
			},
			expBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x06, 0x01, 0xA1, 0xE6},
		},
		{
			name: "Valid instruction with multiple params",
			inst: &instruction{
				id:      0x23,
				command: read,
				params:  []byte{0x84, 0x00, 0x04, 0x00},
			},
			expBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x23, 0x07, 0x00, 0x02, 0x84, 0x00, 0x04,
				0x00, 0xDE, 0xB5},
		},
		{
			name: "Valid instruction with many params",
			inst: &instruction{
				id:      BroadcastID,
				command: fastBRead,
				params:  []byte{0x03, 0x84, 0x00, 0x04, 0x00, 0x07, 0x7C, 0x00, 0x02, 0x00, 0x04, 0x92, 0x00, 0x01, 0x00},
			},
			expBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0xFE, 0x12, 0x00, 0x9A, 0x03, 0x84, 0x00, 0x04, 0x00, 0x07,
				0x7C, 0x00, 0x02, 0x00, 0x04, 0x92, 0x00, 0x01, 0x00, 0xDA, 0x2D},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.inst.packetBytes()
			if tc.expErr != nil {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if !errors.Is(err, tc.expErr) {
					t.Errorf("Expected error %q, got %q", tc.expErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no errors, got %q", err)
			}
			if !reflect.DeepEqual(got, tc.expBytes) {
				t.Errorf("Expected %v, got %v", tc.expBytes, got)
			}
		})
	}
}

func TestParseStatusPacket(t *testing.T) {
	testCases := []struct {
		name        string
		packetBytes []byte
		expErr      error
		expStatus   status
	}{
		{
			name:        "Valid Ping Response",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x07, 0x00, 0x55, 0x00, 0x06, 0x04, 0x26, 0x65, 0x5D},
			expStatus:   status{id: 1, err: nil, params: []byte{0x06, 0x04, 0x26}},
		},
		{
			name:        "Valid Read Response",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0xA6, 0x08, 0x00, 0x55, 0x00, 0xA6, 0x00, 0x00, 0x00, 0xA5, 0xAF},
			expStatus:   status{id: 0xA6, err: nil, params: []byte{0xA6, 0x00, 0x00, 0x00}},
		},
		{
			name:        "Valid Response with Device Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x80, 0xA2, 0x8F},
			expStatus:   status{id: 0x01, err: errDeviceError, params: []byte{}},
		},
		{
			name:        "Valid Response with Result Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x01, 0xA4, 0x8C},
			expStatus:   status{id: 0x01, err: errResultError, params: []byte{}},
		},
		{
			name:        "Valid Response with Instruction Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x02, 0xAE, 0x8C},
			expStatus:   status{id: 0x01, err: errInstructionError, params: []byte{}},
		},
		{
			name:        "Valid Response with CRC Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x03, 0xAB, 0x0C},
			expStatus:   status{id: 0x01, err: errDeviceCRCError, params: []byte{}},
		},
		{
			name:        "Valid Response with Device Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x04, 0xBA, 0x8C},
			expStatus:   status{id: 0x01, err: errDataRangeError, params: []byte{}},
		},
		{
			name:        "Valid Response with Data Length Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x05, 0xBF, 0x0C},
			expStatus:   status{id: 0x01, err: errDataLengthError, params: []byte{}},
		},
		{
			name:        "Valid Response with Data Limit Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x06, 0xB5, 0x0C},
			expStatus:   status{id: 0x01, err: errDataLimitError, params: []byte{}},
		},
		{
			name:        "Valid Response with Access Error",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x04, 0x00, 0x55, 0x07, 0xB0, 0x8C},
			expStatus:   status{id: 0x01, err: errAccessError, params: []byte{}},
		},
		{
			name:        "Packet Too Short",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0xFF, 0x01, 0x00, 0x55, 0x00},
			expErr:      errTruncatedStatus,
		},
		{
			name:        "Packet with Incorrect Instruction",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x07, 0x00, 0x05, 0x00, 0x06, 0x04, 0x26, 0x65, 0x5D},
			expErr:      errMalformedStatus,
		},
		{
			name:        "Packet with Incorrect Length Value",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x08, 0x01, 0x55, 0x00, 0x06, 0x04, 0x26, 0x65, 0x5D},
			expErr:      errInvalidStatusLength,
		},
		{
			name:        "Packet with Invalid CRC bytes",
			packetBytes: []byte{0xFF, 0xFF, 0xFD, 0x00, 0x01, 0x07, 0x00, 0x55, 0x00, 0x06, 0x04, 0x26, 0xA4, 0x8F},
			expErr:      errStatusCRCInvalid,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseStatusPacket(tc.packetBytes)
			if tc.expErr != nil {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if !errors.Is(err, tc.expErr) {
					t.Errorf("Expected error %q, got %q", tc.expErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no errors, got %q", err)
			}
			if got.id != tc.expStatus.id {
				t.Errorf("Expected id to be %d, got %d", tc.expStatus.id, got.id)
			}
			if !errsEqual(got.err, tc.expStatus.err) {
				t.Errorf("Expected err to be %q, got %q", tc.expStatus.err, got.err)
			}
			if !reflect.DeepEqual(got.params, tc.expStatus.params) {
				t.Errorf("Expected params to be %+v, got %+v", tc.expStatus.params, got.params)
			}
		})
	}
}

func errsEqual(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 == nil || err2 == nil {
		return false
	}
	return err1.Error() == err2.Error()
}
