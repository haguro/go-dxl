package protocol

import (
	"bytes"
	"fmt"
	"time"
)

type MockDeviceConfig struct {
	ID                 int
	ModelNumber        int
	FirmwareVer        int
	ControlTable       []byte
	InstructionTimeout time.Duration
	InternalDelay      time.Duration //Additional to the status delay configured in the control table
}

type MockDevice struct {
	buf     bytes.Buffer
	id      byte
	model   uint16
	fw      byte
	ct      []byte
	timeout time.Duration
	delay   time.Duration
	//TODO more configuration to initialise some needed values and to control behaviour (e.g. simulate errors etc)
}

func NewMockDevice(config MockDeviceConfig) *MockDevice {
	b := bytes.Buffer{}
	if config.InstructionTimeout == 0 {
		config.InstructionTimeout = 3000 * time.Microsecond
	}
	return &MockDevice{
		buf:     b,
		id:      byte(config.ID),
		model:   uint16(config.ModelNumber),
		fw:      byte(config.FirmwareVer),
		ct:      config.ControlTable,
		timeout: config.InstructionTimeout,
		delay:   config.InternalDelay,
	}
}

func (d *MockDevice) Read(p []byte) (int, error) {
	return d.buf.Read(p)
}

func (d *MockDevice) Write(p []byte) (int, error) {
	pLen := len(p)
	if pLen < 10 {
		return pLen, nil
	}
	//TODO validate header, length, crc
	id := p[4]
	if id != d.id {
		return pLen, nil
	}
	instLength := uint16(p[5]) + uint16(p[6])<<8
	cmd := p[7]

	errByte := byte(0)
	var params []byte //Assign default value once all instructions (commands) are implemented

	switch cmd {
	case ping:
		params = []byte{byte(d.model), byte(d.model >> 8), d.fw}
	case read:
		if instLength != 7 {
			return pLen, nil
		}
		addr, l := uint16(p[8])+uint16(p[9])<<8, uint16(p[10])+uint16(p[11])<<8
		params = d.ct[addr : addr+l]
	case write, regWrite, action, reset, reboot, clear, backup, syncRead, syncWrite, fastSWrite, bulkRead, bulkWrite, fastBRead:
		panic(fmt.Sprintf("TODO handle instruction %xd", cmd))
	default:
		errByte = 0x02
	}

	length := 4 + uint16(len(params))

	packet := []byte{0xFF, 0xFF, 0xFD, 0x00}
	packet = append(packet, id, byte(length), byte(length>>8), statusCmd, errByte)
	packet = append(packet, params...)
	packet = append(packet, 0, 0)
	updatePacketCRCBytes(packet)

	if d.delay > 0 {
		time.Sleep(d.delay)
	}
	
	d.buf.Write(packet)

	return pLen, nil
}
