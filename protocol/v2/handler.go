package protocol

import (
	"fmt"
	"io"
	"time"
)

const (
	ResetAll                byte = 0xFF // Reset everything in the device control table its default factory value
	ResetAllExceptID        byte = 0x01 // Reset everything except the device ID
	ResetAllExceptIDAndBaud byte = 0x02 // Reset everything except the device ID and the Baud Rate
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
	rw          io.ReadWriter
	readTimeout time.Duration
}

// PingResponse encapsulates the information returned by a ping instruction.
type PingResponse struct {
	ID       byte
	Model    uint16
	Firmware byte
}

// BulkRDescriptor describes the information required to bulk-read data.
type BulkReadDescriptor struct {
	ID     byte   //The ID of the device to read from.
	Addr   uint16 //The starting address to read from.
	Length uint16 //The number of bytes to read.
}

// BulkWDescriptor describes the information required to bulk-write data.
type BulkWriteDescriptor struct {
	ID   byte   //The ID of the device to write to.
	Addr uint16 //The starting address to write to.
	Data []byte //The data to write.
}

// NewHandler creates a new handler for communicating with Dynamixel devices
// with Protocol 2.0 support.
func NewHandler(rw io.ReadWriter, readTimeout time.Duration) *Handler {
	if readTimeout == 0 {
		readTimeout = 20 * time.Millisecond
	}
	return &Handler{
		rw:          rw,
		readTimeout: readTimeout,
	}
}

func (h *Handler) writeInstruction(id, command byte, params ...byte) error {
	inst := &instruction{id, command, params}
	packet, err := inst.packetBytes()
	if err != nil {
		return fmt.Errorf("failed to create instruction packet: %w", err)
	}
	_, err = h.rw.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to write instruction packet bytes: %w", err)
	}
	return nil
}

func (h *Handler) readWithTimeout(b []byte) (int, error) {
	var N int
	timer := time.NewTimer(h.readTimeout)
	defer timer.Stop()
	p := make([]byte, 1)
	for i := 0; i < len(b); i++ {
		for {
			n, err := h.rw.Read(p)
			N += n
			if err != nil {
				if err == io.EOF {
					select {
					case <-timer.C:
						return N, ErrReadTimeout
					default:
						continue
					}
				}
				return N, err
			}
			break
		}
		b[i] = p[0]
	}
	return N, nil
}

func (h *Handler) readStatus() (status, error) {
	var packet []byte

	//Find the header pattern in the stream of bytes
	for {
		b := make([]byte, 1)
		_, err := h.readWithTimeout(b)
		if err != nil {
			return status{}, fmt.Errorf("failed to read status packet header: %w", err)
		}

		packet = append(packet, b[0])
		hl := len(packet)
		if hl >= 4 {
			// Check if the last n bytes match pattern
			if packet[hl-4] == header1 &&
				packet[hl-3] == header2 &&
				packet[hl-2] == header3 &&
				packet[hl-1] == headerR {
				// Header found, break out of loop
				break
			}
			// No match, drop first byte and continue
			packet = packet[1:]
		}
	}

	idLength := make([]byte, 3)
	_, err := h.readWithTimeout(idLength)
	if err != nil {
		return status{}, fmt.Errorf("failed to read status packet ID and Length: %w", err)
	}
	length := uint16(idLength[1]) + uint16(idLength[2])<<8
	// It should be impossible for the length value to be less than 4 bytes (instruction, error, crc(low)
	// and crc(high)).
	// We have to check this again when parsing the packet but we need to stop early if it where to somehow happen.
	if length < 4 {
		return status{}, ErrInvalidStatusLength
	}
	packet = append(packet, idLength...)

	instErrParamsCRC := make([]byte, length) // instruction, error, params and crc bytes
	_, err = h.readWithTimeout(instErrParamsCRC)
	if err != nil {
		return status{}, fmt.Errorf("failed to read status packet instruction, error, params and crc: %w", err)
	}
	packet = append(packet, instErrParamsCRC...)

	return parseStatusPacket(packet)
}

// Ping sends a `ping` instruction to the device with the given ID to check if it is alive and returns the device's
// model number and firmware version.
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

// Read sends a `read` instruction to the device with the given ID to read a given length of data from the device's
// control table starting at the given address.
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

// Write sends a `write` instruction to the device with the given ID to write the given data to the given address of
// the device's control table.
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

// RegWrite sends a `register write` instruction to the device with the given ID to register writing the given data to the
// given address the next time the 'action' instruction is sent to the device.
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

// Action sends an `action` instruction to the device with the given ID to write the data in the previously registered instruction
// (with the `regWrite` instruction) to the device's control table.
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

// Reboot sends a `reboot` instruction to the device with the given ID to reboot the device.
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

// FactoryReset sends a `reset` instruction to the device with the given ID to reset the device's control table
// to its default values.
// One of the following constants can be passed to the `option` parameter:
//   - `ResetAll`: Reset all values to their default values.
//   - `ResetAllExceptID`: Reset all values except the device ID to their default values.
//   - `ResetAllExceptIDAndBaud`: Reset all values except the device ID and baudrate to their default values.
//
// Note that using the `ResetAll` option cannot be used with BroadcastID.
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

// Clear sends a `clear` instruction to the device with the given ID to clear the device's status packet.
// The `option` parameter is maintained for future compatiblity. Only the following constants can be passed
// to the `option` parameter:
// - `ClearMultiRotationPos`: Resets the Present Position value to an absolute value within one rotation (0-4095).lear the status packet.
// Note that this can only be applied when the device is stopped.
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

// ControlTableBackup sends a `backup` instruction to the device with the given ID which can be used to backup or
// restore the device's control table.
// The following constants can be passed to the `option` parameter:
//   - `BackupStore`: Store the current control table in the device's backup area.
//   - `BackupRestore`: Restore the control table from the device's backup area. The device's control table will be
//     overwritten and the device will be rebooted.
//
// Note that this will only work if the device is in Torque OFF mode.
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

// SyncRead sends a `sync read` instruction to the device(s) with the given IDs to read a given length of data from the
// given address from each of the device's control tables.
// Returns a slice of slices of bytes where each inner slice is the data read each the device's control table.
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

// SyncWrite sends a `sync write` instruction to the device(s) with the given IDs to write the given data to the
// given address in each of the device's control tables.
func (h *Handler) SyncWrite(addr, length uint16, data ...byte) error {
	params := []byte{byte(addr), byte(addr >> 8), byte(length), byte(length >> 8)}
	params = append(params, data...)

	if err := h.writeInstruction(BroadcastID, syncWrite, params...); err != nil {
		return fmt.Errorf("failed to send sync write instruction: %w", err)
	}

	return nil
}

// BulkRead sends a `bulk read` instruction to one or more devices. This can read data of different lengths from different
// addresses from different devices.
// Returns a slice of slices of bytes where each inner slice is the data read from the device. The order of the slices
// corresponds to the order of each device ID in the `data` slice.
// Note that each device ID in the `data` can only be used once.
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

// BulkWrite sends a `bulk write` instruction to one or more devices. This can write data of different lengths
// at different addresses to different devices.
// Note that each device ID in the `data` can only be used once.
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

func (h *Handler) FastSyncRead(ids []byte, addr, length uint16) ([][]byte, error) {
	if len(ids) < 1 {
		return nil, fmt.Errorf("fast sync read requires at least one device ID") //TODO error value
	}

	params := []byte{byte(addr), byte(addr >> 8), byte(length), byte(length >> 8)}
	params = append(params, ids...)

	if err := h.writeInstruction(BroadcastID, fastSyncRead, params...); err != nil {
		return nil, fmt.Errorf("failed to send fast sync read instruction: %w", err)
	}

	r, err := h.readStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to read/parse fast sync read status: %w", err)
	}

	if r.err != nil {
		return nil, r.err
	}

	if len(r.params) != int(length)+(len(ids)-1)*(int(length)+4)+1 {
		return nil, ErrUnexpectedParamCount //TODO likely need a seeperate error value here for malformed FSR response
	}

	responses := make([][]byte, len(ids))
	responses[0] = r.params[1 : int(length)+1]
	for i := 1; i < len(ids); i++ {
		responses[i] = r.params[int(length)+5+(i-1)*8 : int(length)+5+(i-1)*8+int(length)]
	}

	return responses, nil
}

// TODO FastBulkRead
// func (h *Handler) FastBulkRead(data []BulkReadDescriptor) ([][]byte, error) {
// 	params := []byte{}
// 	for _, dd := range data {
// 		params = append(params,
// 			dd.ID, byte(dd.Addr), byte(dd.Addr>>8),
// 			byte(dd.Length), byte(dd.Length>>8))
// 	}

// 	if err := h.writeInstruction(BroadcastID, bulkRead, params...); err != nil {
// 		return nil, fmt.Errorf("failed to send bulk read instruction: %w", err)
// 	}
// }
