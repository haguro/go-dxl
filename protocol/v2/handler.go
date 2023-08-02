package protocol

import (
	"fmt"
	"io"
	"log"
)

const (
	NoLogging byte = 0
	LogRead   byte = 1 << iota
	LogWrite
	LogReadWrite = LogRead | LogWrite
)

const (
	ResetAll          byte = 0xFF // Reset everything in the device control table its default factory value
	ResetExceptID     byte = 0x01 // Reset everything except the device ID
	ResetExceptIDBaud byte = 0x02 // Reset everything except the device ID and the Baud Rate
)

const ClearMultiRotationPos byte = 0x01

const (
	BackupStore   byte = 0x01 // Create a store a backup of the device's control table in an internal register.
	BackupRestore byte = 0x02 // Restores an existing device control table backup in an internal register to the current control table.
)

// Handler provides a high level API for interacting with Dynamixel devices
// over a communication interface. It handles constructing protocol packets,
// sending instructions, and parsing status responses.
type Handler struct {
	rw      io.ReadWriter
	logOpts byte
}

// PingResponse encapsulates the information returned by a ping instruction.
type PingResponse struct {
	ID       byte
	Model    uint16
	Firmware byte
}

// BulkRDescriptor describes the information required to bulk-read data
type BulkReadDescriptor struct {
	ID     byte
	Addr   uint16
	Length uint16
}

// BulkWDescriptor describes the information required to bulk-write data
type BulkWriteDescriptor struct {
	ID   byte
	Addr uint16
	Data []byte
}

// NewHandler creates a new handler for communicating with Dynamixel devices
// with Protocol 2.0 support
func NewHandler(rw io.ReadWriter, logPacketOpts byte) *Handler {
	return &Handler{
		rw:      rw,
		logOpts: logPacketOpts,
	}
}

// TODO Thread-safety: Support concurrent use (with mutex?)
func (h *Handler) writeInstruction(id, command byte, params ...byte) error {
	inst := &instruction{id, command, params}
	packet, err := inst.packetBytes()
	if err != nil {
		return fmt.Errorf("failed to create instruction packet: %w", err)
	}
	if h.logOpts&LogWrite != 0 {
		logPacket(packet)
	}
	_, err = h.rw.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to write instruction packet bytes: %w", err)
	}
	return nil
}

// TODO Calling this method should either returns a single status packet if already in the buffer
// or it waits if the buffer is empty until the a time out occurs (with an error). Basically ignore
// EOF errors and implement request timeout mechanism.
// TODO Thread-safety: Support concurrent use (with mutex?)
func (h *Handler) readStatus() (status, error) {
	packet := make([]byte, 7)
	n, err := h.rw.Read(packet)
	if err != nil {
		return status{}, fmt.Errorf("failed to read status packet: %w", err)
	}
	if n < 7 { //Note this won't be needed once wait/timeout is implemented.
		return status{}, ErrTruncatedStatus
	}
	//TODO header pattern scanner? It would theoretically render the initial Flush call useless!
	if packet[0] != 0xFF || packet[1] != 0xFF || packet[2] != 0xFD || packet[3] != 0x00 {
		return status{}, ErrMalformedStatus
	}

	length := uint16(packet[5]) + uint16(packet[6])<<8

	// It should be impossible for the length value to be less than 4 bytes (instruction, error, crc(low)
	// and crc(high)).
	// We have to check this again when parsing the packet but we need to stop early if it where to happen.
	if length < 4 {
		return status{}, ErrInvalidStatusLength
	}

	instErrParamsCRC := make([]byte, length) // instruction, error, params and crc bytes
	n, err = h.rw.Read(instErrParamsCRC)
	if err != nil {
		return status{}, fmt.Errorf("failed to read status packet: %w", err)
	}
	if n < int(length) {
		return status{}, ErrTruncatedStatus
	}

	packet = append(packet, instErrParamsCRC...)

	if h.logOpts&LogRead != 0 {
		logPacket(packet)
	}

	return parseStatusPacket(packet)
}

func (h *Handler) Flush() error {
	_, err := io.Copy(io.Discard, h.rw)
	return err
}

func (h *Handler) Ping(id byte) (PingResponse, error) {
	if err := h.writeInstruction(id, ping); err != nil {
		return PingResponse{}, fmt.Errorf("failed to send ping instruction: %w", err)
	}

	r, err := h.readStatus()
	if err != nil {
		return PingResponse{}, fmt.Errorf("failed to parse ping status: %w", err)
	}

	if r.err != nil {
		return PingResponse{}, r.err
	}

	if len(r.params) != 3 {
		return PingResponse{}, ErrUnexpectedParamCount
	}

	return PingResponse{
		ID:       r.id,
		Model:    uint16(r.params[0]) + uint16(r.params[1])<<8,
		Firmware: r.params[2],
	}, nil
}

func (h *Handler) Read(id byte, addr, length uint16) (data []byte, err error) {
	if id == BroadcastID {
		return nil, ErrNoStatusOnBroadcast
	}
	if err := h.writeInstruction(id, read, byte(addr), byte(addr>>8), byte(length), byte(length>>8)); err != nil {
		return nil, fmt.Errorf("failed to send read instruction: %w", err)
	}

	r, err := h.readStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to read/parse read status: %w", err)
	}

	if r.err != nil {
		return nil, r.err
	}

	if len(r.params) != int(length) {
		return nil, ErrUnexpectedParamCount
	}

	return r.params, nil
}

func (h *Handler) Write(id byte, addr uint16, data ...byte) error {
	params := []byte{byte(addr), byte(addr >> 8)}
	params = append(params, data...)

	if err := h.writeInstruction(id, write, params...); err != nil {
		return fmt.Errorf("failed to send write instruction: %w", err)
	}
	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse write status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}
	return nil
}

func (h *Handler) RegWrite(id byte, addr uint16, data ...byte) error {
	params := []byte{byte(addr), byte(addr >> 8)}
	params = append(params, data...)

	if err := h.writeInstruction(id, regWrite, params...); err != nil {
		return fmt.Errorf("failed to send reg write instruction: %w", err)
	}

	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse reg write status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}
	return nil
}

func (h *Handler) Action(id byte) error {
	if err := h.writeInstruction(id, action); err != nil {
		return fmt.Errorf("failed to send action instruction: %w", err)
	}

	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse action status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}

	return nil
}

func (h *Handler) Reboot(id byte) error {
	if err := h.writeInstruction(id, reboot); err != nil {
		return fmt.Errorf("failed to send reboot instruction: %w", err)
	}

	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse reboot status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}

	return nil
}

func (h *Handler) FactoryReset(id, option byte) error {
	if err := h.writeInstruction(id, reset, option); err != nil {
		return fmt.Errorf("failed to send reset instruction: %w", err)
	}

	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse reset status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}

	return nil
}

func (h *Handler) Clear(id, option byte) error {
	if err := h.writeInstruction(id, clear, option, 0x44, 0x58, 0x4C, 0x22); err != nil {
		return fmt.Errorf("failed to send clear instruction: %w", err)
	}

	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse clear status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}

	return nil
}

func (h *Handler) ControlTableBackup(id byte, option byte) error {
	if err := h.writeInstruction(id, backup, option, 0x43, 0x54, 0x52, 0x4C); err != nil {
		return fmt.Errorf("failed to send backup instruction: %w", err)
	}

	if id != BroadcastID {
		r, err := h.readStatus()
		if err != nil {
			return fmt.Errorf("failed to read/parse backup status: %w", err)
		}
		if r.err != nil {
			return fmt.Errorf("device ID %d returned error: %w", id, r.err)
		}
	}

	return nil
}

func (h *Handler) SyncRead(ids []byte, addr, length uint16) ([][]byte, error) {
	params := []byte{byte(addr), byte(addr >> 8), byte(length), byte(length >> 8)}
	params = append(params, ids...)

	if err := h.writeInstruction(BroadcastID, syncRead, params...); err != nil {
		return nil, fmt.Errorf("failed to send sync read instruction: %w", err)
	}

	var responses [][]byte
	for range ids {
		r, err := h.readStatus()
		if err != nil {
			return nil, fmt.Errorf("failed to read/parse sync read status: %w", err)
		}
		if r.err != nil {
			return nil, r.err
		}
		if len(r.params) != int(length) {
			return nil, ErrUnexpectedParamCount
		}
		responses = append(responses, r.params)
	}

	return responses, nil
}

func (h *Handler) SyncWrite(addr, length uint16, data ...byte) error {
	params := []byte{byte(addr), byte(addr >> 8), byte(length), byte(length >> 8)}
	params = append(params, data...)

	if err := h.writeInstruction(BroadcastID, syncWrite, params...); err != nil {
		return fmt.Errorf("failed to send sync write instruction: %w", err)
	}

	return nil
}

func (h *Handler) BulkRead(data []BulkReadDescriptor) ([][]byte, error) {
	params := []byte{}
	for _, dd := range data {
		params = append(params,
			dd.ID, byte(dd.Addr), byte(dd.Addr>>8),
			byte(dd.Length), byte(dd.Length>>8))
	}

	if err := h.writeInstruction(BroadcastID, bulkRead, params...); err != nil {
		return nil, fmt.Errorf("failed to send bulk read instruction: %w", err)
	}

	var responses [][]byte
	for _, dd := range data {
		r, err := h.readStatus()
		if err != nil {
			return nil, fmt.Errorf("failed to read/parse bulk read status: %w", err)
		}
		if r.err != nil {
			return nil, r.err
		}
		if len(r.params) != int(dd.Length) {
			return nil, ErrUnexpectedParamCount
		}
		responses = append(responses, r.params)
	}
	return responses, nil
}

func (h *Handler) BulkWrite(data []BulkWriteDescriptor) error {
	params := []byte{}
	for _, dd := range data {
		length := len(dd.Data)
		params = append(params, dd.ID)
		params = append(params, byte(dd.Addr), byte(dd.Addr>>8))
		params = append(params, byte(length), byte(length>>8))
		params = append(params, dd.Data...)
	}

	if err := h.writeInstruction(BroadcastID, bulkWrite, params...); err != nil {
		return fmt.Errorf("failed to send bulk write instruction: %w", err)
	}

	return nil
}

// TODO FastSyncRead
// TODO FastBulkRead

func logPacket(packet []byte) {
	n := len(packet)
	inHex := "["
	for i := 0; i < n; i++ {
		inHex += fmt.Sprintf("%02X", packet[i])
		if i < n-1 {
			inHex += "|"
		}
	}
	log.Println(inHex + "]")
}
