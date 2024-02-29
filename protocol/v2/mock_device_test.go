package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"
)

var ErrMockWriteError = errors.New("mock write error")
var ErrMockReadError = errors.New("mock read error")

// Buffer is a thread-safe version of the Readwriter implementation of bytes.Buffer
type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

type DeviceChain struct {
	devices []*MockDevice
	buf     *Buffer
}

func NewDeviceChain(devices ...*MockDevice) *DeviceChain {
	return &DeviceChain{
		devices: devices,
		buf:     &Buffer{},
	}
}

func (c *DeviceChain) Read(p []byte) (int, error) {
	for _, d := range c.devices {
		_, err := io.Copy(c.buf, d)
		if err != nil {
			if err == io.EOF {
				continue
			}
			return 0, fmt.Errorf("failed to read from device chain: %w", err)
		}
	}
	return c.buf.Read(p)
}

func (c *DeviceChain) Write(p []byte) (int, error) {
	for _, d := range c.devices {
		_, err := d.Write(p)
		if err != nil {
			return 0, fmt.Errorf("failed to write to device chain: %w", err)
		}
	}
	return len(p), nil
}

type MockDeviceConfig struct {
	ID                 int
	MidPacketDelay     time.Duration //Simulate delay occuring while writing status packet
	DelayPosition      int
	ProcessingError    int
	ErrorOnRead        bool
	ErrorOnWrite       bool
	SimWrongParamCount bool
}

type MockDevice struct {
	buf             *Buffer
	id              byte
	writeDelay      time.Duration
	delayPos        int
	errorByte       byte
	writeErr        error
	readErr         error
	wrongParamCount bool
	padWithGarbage  bool
}

func NewMockDevice(config MockDeviceConfig) *MockDevice {
	b := Buffer{}
	d := MockDevice{
		buf:             &b,
		id:              byte(config.ID),
		writeDelay:      config.MidPacketDelay,
		delayPos:        config.DelayPosition,
		errorByte:       byte(config.ProcessingError),
		wrongParamCount: config.SimWrongParamCount,
		padWithGarbage:  true, //Always pad status with garbage to simulate potential leftover bytes or noise in channel
	}
	if config.ErrorOnRead {
		d.readErr = ErrMockReadError
	}
	if config.ErrorOnWrite {
		d.writeErr = ErrMockWriteError
	}
	return &d
}

func (d *MockDevice) Read(p []byte) (int, error) {
	if d.readErr != nil {
		return 0, d.readErr
	}
	return d.buf.Read(p)
}

func (d *MockDevice) Write(p []byte) (int, error) {
	if d.writeErr != nil {
		return 0, d.writeErr
	}
	pLen := len(p)
	instID := p[4]
	if instID != BroadcastID && instID != d.id {
		// Not for us. Ignore.
		return pLen, nil
	}

	instLength := uint16(p[5]) + uint16(p[6])<<8
	instruction := p[7]
	instParams := p[8 : 8+instLength-3]

	errByte := d.errorByte
	statusParams := []byte{}
	statusPacket := []byte{}
	length := 4

	switch instruction {
	case ping:
		statusParams = randBytes(3)
	case read:
		l := int(instParams[2]) + int(instParams[3])<<8
		statusParams = randBytes(l)
	case write:
		//No behaivour to mock.
	case regWrite:
		//No behaivour to mock.
	case action:
		//No behaivour to mock.
	case reset:
		//No behaivour to mock.
	case reboot:
		//No behaivour to mock.
	case clear:
		if instParams[0] > 1 ||
			instParams[1] != 0x44 ||
			instParams[2] != 0x58 ||
			instParams[3] != 0x4C ||
			instParams[4] != 0x22 {
			errByte = 1
		}
	case backup:
		if instParams[0] > 2 ||
			instParams[1] != 0x43 ||
			instParams[2] != 0x54 ||
			instParams[3] != 0x52 ||
			instParams[4] != 0x4C {
			errByte = 1
		}
	case syncRead:
		l := int(instParams[2]) + int(instParams[3])<<8
		ids := instParams[4:]
		if !inIDs(ids, d.id) {
			// Nothing for us here. Ignore.
			return pLen, nil
		}
		statusParams = randBytes(l)
	case syncWrite:
		//No behaivour to mock.
	case bulkRead:
		for i := 0; i < len(instParams); i += 5 {
			if instParams[i] == d.id {
				l := int(instParams[i+3]) + int(instParams[i+4])<<8
				statusParams = randBytes(l)
			}
		}
	case bulkWrite:
		//No behaivour to mock.
	case fastSyncRead:
		l := int(instParams[2]) + int(instParams[3])<<8
		ids := instParams[4:]
		if d.id != ids[0] {
			return pLen, nil
		}
		statusParams = append([]byte{d.id}, randBytes(l)...)
		if len(ids) > 1 {
			statusParams = append(statusParams, randBytes(2)...) // TODO: ideally we need to be able to verify the CRC of the EACH of the status packets. For now, we just append two random bytes.
			for k, id := range ids[1:] {
				statusParams = append(statusParams, 0, id)
				statusParams = append(statusParams, randBytes(l)...)
				if k < len(ids)-2 {
					statusParams = append(statusParams, randBytes(2)...) //TODO: Random CRC bytes. See above TODO.
				}
			}
		}

	case fastBulkRead:
		if d.id != instParams[0] {
			return pLen, nil
		}
		l := int(instParams[3]) + int(instParams[4])<<8
		statusParams = append([]byte{d.id}, randBytes(l)...)
		if len(instParams) > 5 {
			statusParams = append(statusParams, randBytes(2)...) // TODO: ideally we need to be able to verify the CRC of the EACH of the status packets. For now, we just append two random bytes.
			for i := 5; i < len(instParams); i += 5 {
				statusParams = append(statusParams, 0, instParams[i])
				l = int(instParams[i+3]) + int(instParams[i+4])<<8
				statusParams = append(statusParams, randBytes(l)...)
				if i < len(instParams)-7 {
					statusParams = append(statusParams, randBytes(2)...) //TODO: Random CRC bytes. See above TODO.
				}
			}
		}

	default:
		errByte = 0x02
	}

	// If the Broadcast ID is used, only Ping, Sync Read and Bulk Read instructions should return status packets
	// see https://emanual.robotis.com/docs/en/dxl/protocol2/#response-policy
	if instID != BroadcastID || instruction == ping ||
		instruction == syncRead || instruction == bulkRead ||
		instruction == fastSyncRead || instruction == fastBulkRead {
		if d.wrongParamCount {
			if len(statusParams) > 1 {
				statusParams = statusParams[:len(statusParams)-1]
			}
			if len(statusParams) <= 1 {
				statusParams = append(statusParams, randBytes(1)...)
			}
		}

		if errByte == 0 {
			length += len(statusParams)
		}
		statusPacket = append(statusPacket, header1, header2, header3, headerR)
		statusPacket = append(statusPacket, instID, byte(length), byte(length>>8), statusCmd, byte(errByte))
		if errByte == 0 {
			statusPacket = append(statusPacket, statusParams...)
		}
		statusPacket = append(statusPacket, 0, 0)
		updatePacketCRCBytes(statusPacket)

		if d.padWithGarbage {
			statusPacket = append(randBytes(rand.Intn(6)), statusPacket...)
			statusPacket = append(statusPacket, randBytes(rand.Intn(6))...)
		}

		if d.writeDelay > 0 {
			d.buf.Write(statusPacket[:d.delayPos])
			go func() {
				time.Sleep(d.writeDelay)
				d.buf.Write(statusPacket[d.delayPos:])
			}()
			return pLen, nil
		}

		d.buf.Write(statusPacket)
	}

	return pLen, nil
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(0xFF))
	}
	return b
}

func inIDs(ids []byte, id byte) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
