package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	ID                 int
	ModelNumber        int
	FirmwareVer        int
	ControlTableRAM    []byte
	InstructionTimeout time.Duration
	InternalDelay      time.Duration //Additional to the status delay configured in the control table
}

type MockDevice struct {
	buf            *bytes.Buffer
	id             byte
	model          uint16
	fw             byte
	ctRAM          []byte
	regWriteBuf    []byte
	regInstruction bool
	CTBackupReg    []byte
	timeout        time.Duration
	delay          time.Duration
	defaultConfig  MockDeviceConfig
	//TODO more configuration to initialise some needed values and to control behaviour (e.g. simulate errors etc)
}

func NewMockDevice(config MockDeviceConfig) *MockDevice {
	b := bytes.Buffer{}
	if config.InstructionTimeout == 0 {
		config.InstructionTimeout = 3000 * time.Microsecond
	}
	d := MockDevice{
		buf:           &b,
		id:            byte(config.ID),
		model:         uint16(config.ModelNumber),
		fw:            byte(config.FirmwareVer),
		timeout:       config.InstructionTimeout,
		delay:         config.InternalDelay,
		defaultConfig: config,
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
	if id != BroadcastID && id != d.id {
		return pLen, nil
	}
	instLength := uint16(p[5]) + uint16(p[6])<<8
	cmd := p[7]
	instParams := p[8 : 8+instLength-3]

	errByte := byte(0)
	var params []byte //Assign default value once all instructions (commands) are implemented

	switch cmd {
	case ping:
		params = []byte{byte(d.model), byte(d.model >> 8), d.fw}
	case read:
		if instLength != 7 {
			return pLen, nil
		}
		addr, l := uint16(instParams[0])+uint16(instParams[1])<<8, uint16(instParams[2])+uint16(instParams[3])<<8
		params = d.ctRAM[addr : addr+l]
	case write:
		if instLength < 6 {
			//TODO return processing error when length is < 6 and larger than 5
			return pLen, nil
		}
		addr := int(instParams[0]) + int(instParams[1])<<8
		for i := 0; i < int(instLength)-5; i++ {
			d.ctRAM[addr+i] = instParams[2+i]
		}
	case regWrite:
		if instLength < 6 {
			//TODO return processing error status when length is < 6 and larger than 5
			return pLen, nil
		}
		d.regWriteBuf = []byte{instParams[0], instParams[1]}
		for i := 0; i < int(instLength)-5; i++ {
			d.regWriteBuf = append(d.regWriteBuf, instParams[2+i])
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
	case reboot:
		for i := range d.ctRAM {
			d.ctRAM[i] = 0
		}
	case clear:
		//TODO verify fixed bytes and option
		switch instParams[0] {
		case ClearMultiRotationPos:
			d.ctRAM[1], d.ctRAM[2], d.ctRAM[3] = d.ctRAM[1]&0x0F, 0, 0
		}
	case reset:
		option := instParams[0]
		if option == ResetAll && id == BroadcastID {
			//Do nothing. See https://emanual.robotis.com/docs/en/dxl/protocol2/#description-5
			return pLen, nil
		}
		for i := range d.ctRAM {
			d.ctRAM[i] = d.defaultConfig.ControlTableRAM[i]
		}
		switch option {
		case ResetAll:
			d.id = byte(d.defaultConfig.ID)
			d.model = uint16(d.defaultConfig.ModelNumber)
			d.fw = byte(d.defaultConfig.FirmwareVer)
		case ResetExceptID, ResetExceptIDBaud: //It's a mock device so BaudRate is irrelevant.
			d.model = uint16(d.defaultConfig.ModelNumber)
			d.fw = byte(d.defaultConfig.FirmwareVer)
		}
	case backup:
		option := instParams[0]
		switch option {
		case BackupStore:
			if d.CTBackupReg == nil {
				d.CTBackupReg = make([]byte, len(d.ctRAM))
			}
			copy(d.CTBackupReg, d.ctRAM)
		case BackupRestore:
			copy(d.ctRAM, d.CTBackupReg)
		}
	case syncRead:
		addr, l := int(instParams[0])+int(instParams[1])<<8, int(instParams[2])+int(instParams[3])<<8
		ids := instParams[4:]
		var idIncluded bool
		for _, devId := range ids {
			if devId == d.id {
				idIncluded = true
				break
			}
		}
		if !idIncluded {
			return pLen, nil
		}
		params = d.ctRAM[addr : addr+l]
	case syncWrite:
		addr, l := int(instParams[0])+int(instParams[1])<<8, int(instParams[2])+int(instParams[3])<<8
		var data []byte
		dataToWrite := instParams[4:]
		for i := 0; i < len(dataToWrite); i += l + 1 {
			if dataToWrite[i] == d.id {
				for j := 0; j < l; j++ {
					data = append(data, dataToWrite[i+j+1])
				}
			}
		}
		for i, v := range data {
			d.ctRAM[addr+i] = v
		}
	case bulkRead:
		if instLength < 8 {
			//No data at all which means no IDs. So will never be matched by any device.
			return pLen, nil
		}
		for i := 0; i < len(instParams); i += 5 {
			if instParams[i] == d.id {
				addr, l := int(instParams[i+1])+int(instParams[i+2])<<8, int(instParams[i+3])+int(instParams[i+4])<<8
				params = d.ctRAM[addr : addr+l]
				fmt.Println(params)
			}
		}
	case bulkWrite:
		if instLength < 8 {
			//No data at all which means no IDs. So will never be matched by any device.
			return pLen, nil
		}
		for i := 0; i < len(instParams); {
			addr, l := int(instParams[i+1])+int(instParams[i+2])<<8, int(instParams[i+3])+int(instParams[i+4])<<8
			if instParams[i] == d.id {
				for j := 0; j < l; j++ {
					d.ctRAM[addr+j] = instParams[5+i+j]
				}
			}
			i += 5 + l
		}
	case fastSyncRead, fastBulkRead:
		panic(fmt.Sprintf("Instruction %x is not implemented", cmd))
	default:
		errByte = 0x02
	}

	time.Sleep(d.delay)

	//TODO Support return level values
	if id != BroadcastID || cmd == ping || cmd == syncRead || cmd == bulkRead {
		length := 4 + uint16(len(params))
		statusPacket := []byte{header1, header2, header3, headerR}
		statusPacket = append(statusPacket, id, byte(length), byte(length>>8), statusCmd, errByte)
		statusPacket = append(statusPacket, params...)
		statusPacket = append(statusPacket, 0, 0)
		updatePacketCRCBytes(statusPacket)

		d.buf.Write(statusPacket)
	}

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
