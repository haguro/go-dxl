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
	ControlTableRAM    []byte
	InstructionTimeout time.Duration
	InternalDelay      time.Duration //Additional to the status delay configured in the control table
}

type MockDevice struct {
	buf            bytes.Buffer
	id             byte
	model          uint16
	fw             byte
	ctRAM          []byte
	regWriteBuf    []byte
	regInstruction bool
	timeout        time.Duration
	delay          time.Duration
	//TODO more configuration to initialise some needed values and to control behaviour (e.g. simulate errors etc)
}

func NewMockDevice(config MockDeviceConfig) *MockDevice {
	b := bytes.Buffer{}
	if config.InstructionTimeout == 0 {
		config.InstructionTimeout = 3000 * time.Microsecond
	}
	d := MockDevice{
		buf:     b,
		id:      byte(config.ID),
		model:   uint16(config.ModelNumber),
		fw:      byte(config.FirmwareVer),
		timeout: config.InstructionTimeout,
		delay:   config.InternalDelay,
	}
	d.ctRAM = make([]byte, len(config.ControlTableRAM))
	copy(d.ctRAM, config.ControlTableRAM)
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
		params = d.ctRAM[addr : addr+l]
	case write:
		if instLength < 6 {
			//TODO return processing error when length is < 6 and larger than 5
			return pLen, nil
		}
		addr := int(p[8]) + int(p[9])<<8
		for i := 0; i < int(instLength)-5; i++ {
			d.ctRAM[addr+i] = p[10+i]
		}
	case regWrite:
		if instLength < 6 {
			//TODO return processing error status when length is < 6 and larger than 5
			return pLen, nil
		}
		d.regWriteBuf = []byte{p[8], p[9]}
		for i := 0; i < int(instLength)-5; i++ {
			d.regWriteBuf = append(d.regWriteBuf, p[10+i])
		}
		d.regInstruction = true
	case action:
		if len(d.regWriteBuf) < 2 || !d.regInstruction {
			errByte = 0x02
			break
		}
		addr := int(d.regWriteBuf[0]) + int(d.regWriteBuf[1])<<8
		for i := 0; i < len(d.regWriteBuf)-2; i++ {
			d.ctRAM[addr+i] = d.regWriteBuf[2+i]
		}
		d.regWriteBuf = []byte{}
		d.regInstruction = false
		panic(fmt.Sprintf("TODO handle instruction %x", cmd))
	default:
		errByte = 0x02
	}

	length := 4 + uint16(len(params))

	statusPacket := []byte{header1, header2, header3, headerR}
	statusPacket = append(statusPacket, id, byte(length), byte(length>>8), statusCmd, errByte)
	statusPacket = append(statusPacket, params...)
	statusPacket = append(statusPacket, 0, 0)
	updatePacketCRCBytes(statusPacket)

	time.Sleep(d.delay)
	d.buf.Write(statusPacket)

	return pLen, nil
}

func (d *MockDevice) InspectRegWriteBuffer() []byte {
	return d.regWriteBuf
}

func (d *MockDevice) InspectControlTable(index, length int) []byte {
	if index+length > len(d.ctRAM) {
		return nil
	}
	return d.ctRAM[index : index+length]
}
