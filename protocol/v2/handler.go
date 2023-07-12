package protocol

import (
	"fmt"
	"io"
)

type Handler struct {
	rw io.ReadWriter
}

type PingResponse struct {
	ID       byte
	Model    uint16
	Firmware byte
}

func NewHandler(rw io.ReadWriter) *Handler {
	return &Handler{
		rw: rw,
	}
}

// TODO Thread-safety: Support concurrent use (with mutex?)
func (h *Handler) writeInstruction(id, command byte, params ...byte) error {
	inst := &instruction{id, command, params}
	packet, err := inst.packetBytes()
	if err != nil {
		return fmt.Errorf("failed to write instruction packet bytes: %w", err)
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
		return status{}, errTruncatedStatus
	}
	//TODO header pattern scanner? It would theoretically render the initial Flush call useless!
	if packet[0] != 0xFF || packet[1] != 0xFF || packet[2] != 0xFD || packet[3] != 0x00 {
		return status{}, errMalformedStatus
	}

	length := uint16(packet[5]) + uint16(packet[6])<<8

	// It should be impossible for the length value to be less than 4 bytes (instruction, error, crc(low)
	// and crc(high)).
	// We have to check this again when parsing the packet but we need to stop early if it should happen.
	if length < 4 {
		return status{}, errInvalidStatusLength
	}

	instErrParamsCRC := make([]byte, length) // instruction, error, params and crc bytes
	n, err = h.rw.Read(instErrParamsCRC)
	if err != nil {
		return status{}, fmt.Errorf("failed to read status packet: %w", err)
	}
	if n < int(length) {
		return status{}, errTruncatedStatus
	}

	packet = append(packet, instErrParamsCRC...)

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
		return PingResponse{}, errUnexpectedParamCount
	}

	return PingResponse{
		ID:       r.id,
		Model:    uint16(r.params[0]) + uint16(r.params[1])<<8,
		Firmware: r.params[2],
	}, nil
}

func (h *Handler) Read(id byte, addr, length uint16) (data []byte, err error) {
	if id == BroadcastID {
		return nil, errNoStatusOnBroadcast
	}
	if err := h.writeInstruction(id, read, byte(addr), byte(addr>>8), byte(length), byte(length>>8)); err != nil {
		return nil, fmt.Errorf("failed to send ping instruction: %w", err)
	}

	r, err := h.readStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ping status: %w", err)
	}

	if r.err != nil {
		return nil, r.err
	}

	if len(r.params) != int(length) {
		return nil, errUnexpectedParamCount
	}

	return r.params, nil
}