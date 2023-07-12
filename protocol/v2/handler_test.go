package protocol_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/haguro/go-dxl/protocol/v2"
)

var deviceConfig = protocol.MockDeviceConfig{
	ID:           0x99,
	ModelNumber:  0x0424,
	FirmwareVer:  0x2F,
	ControlTable: []byte{0x32, 0x14, 0xF0, 0xE9, 0xA9, 0x7C},
}

func TestFlush(t *testing.T) {
	b := bytes.NewBuffer(make([]byte, 10))
	h := protocol.NewHandler(b)

	if err := h.Flush(); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if b.Len() != 0 {
		t.Errorf("Expected buffer to be empty, got %d elements", b.Len())
	}
}

func TestPing(t *testing.T) {
	d := protocol.NewMockDevice(deviceConfig)
	h := protocol.NewHandler(d)

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
	h := protocol.NewHandler(d)
	addr, length := 3, 2

	got, err := h.Read(byte(deviceConfig.ID), uint16(addr), uint16(length))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := deviceConfig.ControlTable[addr : addr+length]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}
