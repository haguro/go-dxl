package protocol_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/haguro/go-dxl/protocol/v2"
)

func TestPing(t *testing.T) {
	var testCases = []struct {
		name            string
		packetDelay     time.Duration
		delayPosition   int
		processingError int
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		expectErr       error
	}{
		{
			name: "No errors, whole packet",
		},
		{
			name:          "No errors, delayed packet start",
			packetDelay:   5 * time.Millisecond,
			delayPosition: 0,
		},
		{
			name:          "No errors, mid-packet delay",
			packetDelay:   5 * time.Millisecond,
			delayPosition: 6,
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:            "Wrong Status Param Count",
			wrongParamCount: true,
			expectErr:       protocol.ErrUnexpectedParamCount,
		},
		{
			name:          "Read Timeout Error, long initial delay",
			packetDelay:   15 * time.Millisecond,
			delayPosition: 0,
			expectErr:     protocol.ErrReadTimeout,
		},
		{
			name:          "Read Timeout Error, long mid-packet delay",
			packetDelay:   15 * time.Millisecond,
			delayPosition: 3,
			expectErr:     protocol.ErrReadTimeout,
		},
	}
	deviceID := 0x99
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:                 deviceID,
				MidPacketDelay:     tc.packetDelay,
				DelayPosition:      tc.delayPosition,
				ProcessingError:    tc.processingError,
				ErrorOnRead:        tc.errOnRead,
				ErrorOnWrite:       tc.errOnWrite,
				SimWrongParamCount: tc.wrongParamCount,
			})
			h := protocol.NewHandler(d, 10*time.Millisecond)

			_, err := h.Ping(byte(deviceID))
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestRead(t *testing.T) {
	var testCases = []struct {
		name            string
		deviceID        byte
		processingError int
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		packetDelay     time.Duration
		delayPosition   int
		expectErr       error
	}{
		{
			name:     "No errors",
			deviceID: 0xEA,
		},
		{
			name:          "No errors, delayed packet start",
			deviceID:      0xEA,
			packetDelay:   10 * time.Millisecond,
			delayPosition: 0,
		},
		{
			name:          "No errors, mid-packet delay",
			deviceID:      0xEA,
			packetDelay:   10 * time.Millisecond,
			delayPosition: 6,
		},
		{
			name:      "ReadStatus error with Broadcast ID",
			deviceID:  protocol.BroadcastID,
			expectErr: protocol.ErrNoStatusOnBroadcast,
		},
		{
			name:            "Device Error",
			deviceID:        0xEA,
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			deviceID:  0xEA,
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			deviceID:   0xEA,
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:            "Wrong Status Param Count",
			deviceID:        0xEA,
			wrongParamCount: true,
			expectErr:       protocol.ErrUnexpectedParamCount,
		},
		{
			name:          "Read Timeout Error, long initial delay",
			deviceID:      0xEA,
			packetDelay:   40 * time.Millisecond,
			delayPosition: 0,
			expectErr:     protocol.ErrReadTimeout,
		},
		{
			name:          "Read Timeout Error, long mid-packet delay",
			deviceID:      0xEA,
			packetDelay:   50 * time.Millisecond,
			delayPosition: 6,
			expectErr:     protocol.ErrReadTimeout,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:                 int(tc.deviceID),
				MidPacketDelay:     tc.packetDelay,
				DelayPosition:      tc.delayPosition,
				ProcessingError:    tc.processingError,
				ErrorOnRead:        tc.errOnRead,
				ErrorOnWrite:       tc.errOnWrite,
				SimWrongParamCount: tc.wrongParamCount,
			})
			h := protocol.NewHandler(d, 0)
			addr, length := 3, 8

			got, err := h.Read(tc.deviceID, uint16(addr), uint16(length))
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}

			if len(got) != length {
				t.Errorf("Expected %d bytes, got %d", length, len(got))
			}
		})
	}
}

func TestWrite(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		packetDelay     time.Duration
		delayPosition   int
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:          "No errors, delayed packet start",
			packetDelay:   50 * time.Millisecond,
			delayPosition: 0,
		},
		{
			name:          "No errors, mid-packet delay",
			packetDelay:   50 * time.Millisecond,
			delayPosition: 6,
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:          "Read Timeout Error, long initial delay",
			packetDelay:   110 * time.Millisecond,
			delayPosition: 0,
			expectErr:     protocol.ErrReadTimeout,
		},
		{
			name:          "Read Timeout Error, long mid-packet delay",
			packetDelay:   110 * time.Millisecond,
			delayPosition: 9,
			expectErr:     protocol.ErrReadTimeout,
		},
	}
	deviceID := 0x7A
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				MidPacketDelay:  tc.packetDelay,
				DelayPosition:   tc.delayPosition,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 100*time.Millisecond)
			addr := 2
			data := []byte{0xF1, 0xF2}
			err := h.Write(byte(deviceID), uint16(addr), data...)
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestRegWrite(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	deviceID := 0x7A
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 0)
			addr := 2
			data := []byte{0xF1, 0xF2}
			err := h.RegWrite(byte(deviceID), uint16(addr), data...)
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestAction(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	deviceID := 0x9D
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 0)
			err := h.Action(byte(deviceID))
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestReboot(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	deviceID := 0xD0
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 0)
			err := h.Reboot(byte(deviceID))
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestClear(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	deviceID := 0x0A
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 0)
			err := h.Clear(byte(deviceID), protocol.ClearMultiRotationPos)
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestFactoryReset(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	deviceID := 0xA7
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 0)
			err := h.FactoryReset(byte(deviceID), protocol.ResetAllExceptID)
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestControlTableBackup(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	deviceID := 0x7C
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := protocol.NewMockDevice(protocol.MockDeviceConfig{
				ID:              deviceID,
				ProcessingError: tc.processingError,
				ErrorOnRead:     tc.errOnRead,
				ErrorOnWrite:    tc.errOnWrite,
			})
			h := protocol.NewHandler(d, 0)
			err := h.ControlTableBackup(byte(deviceID), protocol.BackupStore)
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}

			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestSyncRead(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:            "Wrong Status Param Count",
			wrongParamCount: true,
			expectErr:       protocol.ErrUnexpectedParamCount,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config1 := protocol.MockDeviceConfig{
				ID: 0xB5,
			}
			config2 := protocol.MockDeviceConfig{
				ID: 0xB2,
			}
			config3 := protocol.MockDeviceConfig{
				ID:                 0xB3,
				ProcessingError:    tc.processingError,
				ErrorOnRead:        tc.errOnRead,
				ErrorOnWrite:       tc.errOnWrite,
				SimWrongParamCount: tc.wrongParamCount,
			}
			d1 := protocol.NewMockDevice(config1)
			d2 := protocol.NewMockDevice(config2)
			d3 := protocol.NewMockDevice(config3)
			c := protocol.NewDeviceChain(d1, d2, d3)
			h := protocol.NewHandler(c, 0)

			addr, length := 51, 12
			ids := []byte{byte(config1.ID), byte(config2.ID), byte(config3.ID)}
			got, err := h.SyncRead(ids, uint16(addr), uint16(length))

			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}
			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
			if len(got) != 3 {
				t.Errorf("Expected 3 responses, got %d", len(got))
			}
			for i, v := range got {
				if len(v) != length {
					t.Errorf("Expected %d bytes from response %d, got %d", length, i+1, len(v))
				}
			}
		})
	}
}

func TestSyncWrite(t *testing.T) {
	var testCases = []struct {
		name       string
		errOnRead  bool
		errOnWrite bool
		expectErr  error
	}{
		{
			name: "No errors",
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config1 := protocol.MockDeviceConfig{
				ID: 0x5A,
			}
			config2 := protocol.MockDeviceConfig{
				ID:           int(0x5B),
				ErrorOnRead:  tc.errOnRead,
				ErrorOnWrite: tc.errOnWrite,
			}
			config3 := protocol.MockDeviceConfig{
				ID: 0x5C,
			}
			d1 := protocol.NewMockDevice(config1)
			d2 := protocol.NewMockDevice(config2)
			d3 := protocol.NewMockDevice(config3)
			c := protocol.NewDeviceChain(d1, d2, d3)
			h := protocol.NewHandler(c, 0)

			addr := 4
			data1 := []byte{0xF1, 0xF2}
			data2 := []byte{0xA7, 0xA8}
			data3 := []byte{0x21, 0x43}
			data := []byte{byte(config1.ID)}
			data = append(data, data1...)
			data = append(data, byte(config2.ID))
			data = append(data, data2...)
			data = append(data, byte(config3.ID))
			data = append(data, data3...)

			err := h.SyncWrite(uint16(addr), 2, data...)

			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}
			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestBulkRead(t *testing.T) {
	var testCases = []struct {
		name            string
		deviceID        byte
		processingError int
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:            "Wrong Status Param Count",
			wrongParamCount: true,
			expectErr:       protocol.ErrUnexpectedParamCount,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config1 := protocol.MockDeviceConfig{
				ID: 0x5A,
			}
			config2 := protocol.MockDeviceConfig{
				ID: 0x5B,
			}
			config3 := protocol.MockDeviceConfig{
				ID:                 0x5C,
				ProcessingError:    tc.processingError,
				ErrorOnRead:        tc.errOnRead,
				ErrorOnWrite:       tc.errOnWrite,
				SimWrongParamCount: tc.wrongParamCount,
			}
			d1 := protocol.NewMockDevice(config1)
			d2 := protocol.NewMockDevice(config2)
			d3 := protocol.NewMockDevice(config3)
			c := protocol.NewDeviceChain(d1, d2, d3)
			l := protocol.NewPacketLogger(c, protocol.LogReadWrite, io.Discard)
			h := protocol.NewHandler(l, 0)

			brDesc := []protocol.BulkReadDescriptor{
				{
					ID:     byte(config1.ID),
					Addr:   1,
					Length: 4,
				},
				{
					ID:     byte(config2.ID),
					Addr:   4,
					Length: 10,
				},
				{
					ID:     byte(config3.ID),
					Addr:   2,
					Length: 22,
				},
			}
			got, err := h.BulkRead(brDesc)

			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}
			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}

			if len(got) != len(brDesc) {
				t.Fatalf("Expected all %d devices to return status, only %d did", len(brDesc), len(got))
			}
			for i, v := range got {
				if len(v) != int(brDesc[i].Length) {
					t.Errorf("Expected %d bytes from response %d, got %d", int(brDesc[i].Length), i+1, len(v))
				}
			}
		})
	}
}

func TestBulkWrite(t *testing.T) {
	var testCases = []struct {
		name            string
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config1 := protocol.MockDeviceConfig{
				ID: 0x5A,
			}
			config2 := protocol.MockDeviceConfig{
				ID:           0x5B,
				ErrorOnRead:  tc.errOnRead,
				ErrorOnWrite: tc.errOnWrite,
			}
			config3 := protocol.MockDeviceConfig{
				ID: 0x5C,
			}
			d1 := protocol.NewMockDevice(config1)
			d2 := protocol.NewMockDevice(config2)
			d3 := protocol.NewMockDevice(config3)
			c := protocol.NewDeviceChain(d1, d2, d3)
			h := protocol.NewHandler(c, 0)

			bwDesc := []protocol.BulkWriteDescriptor{
				{
					ID:   byte(config1.ID),
					Addr: 2,
					Data: []byte{0x01, 0x02, 0x03},
				},
				{
					ID:   byte(config2.ID),
					Addr: 4,
					Data: []byte{0x04, 0x05},
				},
				{
					ID:   byte(config3.ID),
					Addr: 5,
					Data: []byte{0x06},
				},
			}
			err := h.BulkWrite(bwDesc)
			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}
			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
		})
	}
}

func TestFastSyncRead(t *testing.T) {
	var testCases = []struct {
		name            string
		processingError int
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:            "Wrong Status Param Count",
			wrongParamCount: true,
			expectErr:       protocol.ErrUnexpectedParamCount,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config1 := protocol.MockDeviceConfig{
				ID:                 0x03,
				ProcessingError:    tc.processingError,
				SimWrongParamCount: tc.wrongParamCount,
				ErrorOnWrite:       tc.errOnWrite,
				ErrorOnRead:        tc.errOnRead,
			}
			config2 := protocol.MockDeviceConfig{
				ID: 0x07,
			}
			config3 := protocol.MockDeviceConfig{
				ID: 0x04,
			}
			d1 := protocol.NewMockDevice(config1)
			d2 := protocol.NewMockDevice(config2)
			d3 := protocol.NewMockDevice(config3)
			c := protocol.NewDeviceChain(d1, d2, d3)
			h := protocol.NewHandler(c, 0)

			addr, length := 0x84, 4
			ids := []byte{byte(config1.ID), byte(config2.ID), byte(config3.ID)}
			got, err := h.FastSyncRead(ids, uint16(addr), uint16(length))

			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}
			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}
			if len(got) != 3 {
				t.Errorf("Expected 3 responses, got %d", len(got))
			}
			for i, v := range got {
				if len(v) != length {
					t.Errorf("Expected %d bytes from response %d, got %d", length, i+1, len(v))
				}
			}
		})
	}
}

func TestFastBulkRead(t *testing.T) {
	var testCases = []struct {
		name            string
		deviceID        byte
		processingError int
		errOnRead       bool
		errOnWrite      bool
		wrongParamCount bool
		expectErr       error
	}{
		{
			name: "No errors",
		},
		{
			name:            "Device Error",
			processingError: 0x80,
			expectErr:       protocol.ErrDeviceError,
		},
		{
			name:      "Read Error",
			errOnRead: true,
			expectErr: protocol.ErrMockReadError,
		},
		{
			name:       "Write Error",
			errOnWrite: true,
			expectErr:  protocol.ErrMockWriteError,
		},
		{
			name:            "Wrong Status Param Count",
			wrongParamCount: true,
			expectErr:       protocol.ErrUnexpectedParamCount,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config1 := protocol.MockDeviceConfig{
				ID:                 0x03,
				ProcessingError:    tc.processingError,
				ErrorOnRead:        tc.errOnRead,
				ErrorOnWrite:       tc.errOnWrite,
				SimWrongParamCount: tc.wrongParamCount,
			}
			config2 := protocol.MockDeviceConfig{
				ID: 0x07,
			}
			config3 := protocol.MockDeviceConfig{
				ID: 0x04,
			}
			d1 := protocol.NewMockDevice(config1)
			d2 := protocol.NewMockDevice(config2)
			d3 := protocol.NewMockDevice(config3)
			c := protocol.NewDeviceChain(d1, d2, d3)
			h := protocol.NewHandler(c, 0)

			brDesc := []protocol.BulkReadDescriptor{
				{
					ID:     byte(config1.ID),
					Addr:   0x0084,
					Length: 4,
				},
				{
					ID:     byte(config2.ID),
					Addr:   0x007C,
					Length: 2,
				},
				{
					ID:     byte(config3.ID),
					Addr:   0x0092,
					Length: 1,
				},
			}
			got, err := h.FastBulkRead(brDesc)

			if err != nil {
				if tc.expectErr == nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !errors.Is(err, tc.expectErr) {
					t.Errorf("Expected error of %q but got type %q", tc.expectErr, err)
				}
				return
			}
			if tc.expectErr != nil {
				t.Errorf("Expected error but got none")
			}

			if len(got) != len(brDesc) {
				t.Fatalf("Expected all %d devices to return status, only %d did", len(brDesc), len(got))
			}
			for i, v := range got {
				if len(v) != int(brDesc[i].Length) {
					t.Errorf("Expected %d bytes from response %d, got %d", int(brDesc[i].Length), i+1, len(v))
				}
			}
		})
	}
}
