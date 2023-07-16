package protocol_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/haguro/go-dxl/protocol/v2"
)

var deviceConfig = protocol.MockDeviceConfig{
	ID:              0x99,
	ModelNumber:     0x424,
	FirmwareVer:     0x2F,
	ControlTableRAM: []byte{0x32, 0x14, 0xF0, 0xE9, 0xA9, 0x7C},
}

func TestFlush(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 10))
	h := protocol.NewHandler(b, protocol.NoLogging)

	if err := h.Flush(); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if b.Len() != 0 {
		t.Errorf("Expected buffer to be empty, got %d elements", b.Len())
	}
}

func TestPing(t *testing.T) {
	d := protocol.NewMockDevice(deviceConfig)
	h := protocol.NewHandler(d, protocol.NoLogging)

	got, err := h.Ping(byte(deviceConfig.ID))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := protocol.PingResponse{
		ID:       byte(deviceConfig.ID),
		Model:    uint16(deviceConfig.ModelNumber),
		Firmware: byte(deviceConfig.FirmwareVer),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestRead(t *testing.T) {
	d := protocol.NewMockDevice(deviceConfig)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr, length := 3, 2

	got, err := h.Read(byte(deviceConfig.ID), uint16(addr), uint16(length))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := deviceConfig.ControlTableRAM[addr : addr+length]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestWrite(t *testing.T) {
	d := protocol.NewMockDevice(deviceConfig)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 4
	data := []byte{0xF1, 0xF2}
	if err := h.Write(byte(deviceConfig.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	got := d.InspectControlTable(addr, len(data))
	if !reflect.DeepEqual(got, data) {
		t.Errorf("Expected %+v, got %+v", data, got)
	}
}

func TestRegWrite(t *testing.T) {
	d := protocol.NewMockDevice(deviceConfig)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 4
	data := []byte{0xF1, 0xF2}
	if err := h.RegWrite(byte(deviceConfig.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := append([]byte{byte(addr), byte(addr >> 8)}, data...)
	got := d.InspectRegWriteBuffer()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected RegWrite to not change control table, got %+v", got)
	}
}

func TestAction(t *testing.T) {
	d := protocol.NewMockDevice(deviceConfig)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 4
	data := []byte{0xF1, 0xF2}
	if err := h.RegWrite(byte(deviceConfig.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.Action(byte(deviceConfig.ID)); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	got := d.InspectControlTable(addr, len(data))
	if !reflect.DeepEqual(got, data) {
		t.Errorf("Expected %+v, got %+v", data, got)
	}
}
