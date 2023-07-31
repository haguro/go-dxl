package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"time"
)

type DeviceChain struct {
	devices []*MockDevice
	buf     *bytes.Buffer
}

func NewDeviceChain(devices ...*MockDevice) *DeviceChain {
	return &DeviceChain{
		devices: devices,
		buf:     &bytes.Buffer{},
	}
}

func (c *DeviceChain) Read(p []byte) (int, error) {
	for _, d := range c.devices {
		_, err := io.Copy(c.buf, d)
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return 0, fmt.Errorf("failed to read from device chain: %w", err)
		}
	}
	return c.buf.Read(p)
}

func (c *DeviceChain) Write(p []byte) (int, error) {
	var t int
	for _, d := range c.devices {
		n, err := d.Write(p)
		if err != nil {
			return 0, fmt.Errorf("failed to write to device chain: %w", err)
		}
		t += n
	}
	return t, nil
}

type MockDeviceConfig struct {
	ID            int
	InternalDelay time.Duration //Additional to the status delay configured in the control table
}

type MockDevice struct {
	buf   *bytes.Buffer
	id    byte
	delay time.Duration
}

func NewMockDevice(config MockDeviceConfig) *MockDevice {
	b := bytes.Buffer{}
	d := MockDevice{
		buf:   &b,
		id:    byte(config.ID),
		delay: config.InternalDelay,
	}
	return &d
}

func (d *MockDevice) Read(p []byte) (int, error) {
	return d.buf.Read(p)
}

// TODO this assumes that the entire instruction packet is written in one go.
// We should handle partial writes and buffer the data until the entire packet is received or d.timeout is reached/
// This might be useful: https://emanual.robotis.com/docs/en/dxl/protocol2/#processing-order-of-reception
func (d *MockDevice) Write(p []byte) (int, error) {
	pLen := len(p)
	instID := p[4]
	if instID != BroadcastID && instID != d.id {
		// Not for us. Ignore.
		return pLen, nil
	}

	time.Sleep(d.delay)

	instLength := uint16(p[5]) + uint16(p[6])<<8
	instruction := p[7]
	instParams := p[8 : 8+instLength-3]

	errByte := 0
	statusParams := []byte{}

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
	case fastSyncRead, fastBulkRead:
		panic(fmt.Sprintf("Instruction %x is not implemented", instruction))
	default:
		errByte = 0x02
	}

	// If the Broadcast ID is used, only Ping, Sync Read and Bulk Read instructions should return status packets
	// see https://emanual.robotis.com/docs/en/dxl/protocol2/#response-policy
	if instID != BroadcastID || instruction == ping || instruction == syncRead || instruction == bulkRead {
		length := 4 + uint16(len(statusParams))
		statusPacket := []byte{header1, header2, header3, headerR}
		statusPacket = append(statusPacket, instID, byte(length), byte(length>>8), statusCmd, byte(errByte))
		statusPacket = append(statusPacket, statusParams...)
		statusPacket = append(statusPacket, 0, 0)
		updatePacketCRCBytes(statusPacket)

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
