package protocol_test

import (
	"bytes"
	"testing"

	"github.com/haguro/go-dxl/protocol/v2"
)

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
	config := protocol.MockDeviceConfig{
		ID: 0x99,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)

	_, err := h.Ping(byte(config.ID))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestRead(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0x01,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr, length := 3, 8

	got, err := h.Read(byte(config.ID), uint16(addr), uint16(length))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if len(got) != length {
		t.Errorf("Expected %d bytes, got %d", length, len(got))
	}
}

func TestWrite(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0x7A,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 2
	data := []byte{0xF1, 0xF2}
	if err := h.Write(byte(config.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestRegWrite(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0xA9,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 2
	data := []byte{0xF1, 0xF2}
	if err := h.RegWrite(byte(config.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestAction(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0x9D,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.Action(byte(config.ID)); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestReboot(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0xD0,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.Reboot(byte(config.ID)); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestClear(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0x0A,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.Clear(byte(config.ID), protocol.ClearMultiRotationPos); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestFactoryReset(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0xAB,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.Clear(byte(config.ID), protocol.ClearMultiRotationPos); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.FactoryReset(byte(config.ID), protocol.ResetExceptID); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestControlTableBackup(t *testing.T) {
	config := protocol.MockDeviceConfig{
		ID: 0xB5,
	}
	d := protocol.NewMockDevice(config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.ControlTableBackup(byte(config.ID), protocol.BackupStore); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.ControlTableBackup(byte(config.ID), protocol.BackupRestore); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestSyncRead(t *testing.T) {
	config1 := protocol.MockDeviceConfig{
		ID: 0xB5,
	}
	config2 := protocol.MockDeviceConfig{
		ID: 0xB2,
	}
	config3 := protocol.MockDeviceConfig{
		ID: 0xB3,
	}
	d1 := protocol.NewMockDevice(config1)
	d2 := protocol.NewMockDevice(config2)
	d3 := protocol.NewMockDevice(config3)
	c := protocol.NewDeviceChain(d1, d2, d3)
	h := protocol.NewHandler(c, protocol.NoLogging)

	addr, length := 51, 12
	ids := []byte{byte(config1.ID), byte(config2.ID), byte(config3.ID)}
	got, err := h.SyncRead(ids, uint16(addr), uint16(length))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if len(got) != 3 {
		t.Errorf("Expected 3 responses, got %d", len(got))
	}
	for i, v := range got {
		if len(v) != length {
			t.Errorf("Expected %d bytes from response %d, got %d", length, i+1, len(v))
		}
	}

}

func TestSyncWrite(t *testing.T) {
	config1 := protocol.MockDeviceConfig{
		ID: 0x5A,
	}
	config2 := protocol.MockDeviceConfig{
		ID: 0x5B,
	}
	config3 := protocol.MockDeviceConfig{
		ID: 0x5C,
	}
	d1 := protocol.NewMockDevice(config1)
	d2 := protocol.NewMockDevice(config2)
	d3 := protocol.NewMockDevice(config3)
	c := protocol.NewDeviceChain(d1, d2, d3)
	h := protocol.NewHandler(c, protocol.NoLogging)

	addr := 4
	data1 := []byte{0xF1, 0xF2}
	data2 := []byte{0xA7, 0xA8}
	data3 := []byte{0x21, 0x43}
	data := []byte{byte(config1.ID)}
	data = append(data, data1...)
	data = append(data, byte(config2.ID))
	data = append(data, data2...)
	data = append(data, byte(config3.ID))
	data = append(data, data3...)

	if err := h.SyncWrite(uint16(addr), 2, data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestBulkRead(t *testing.T) {
	config1 := protocol.MockDeviceConfig{
		ID: 0x5A,
	}
	config2 := protocol.MockDeviceConfig{
		ID: 0x5B,
	}
	config3 := protocol.MockDeviceConfig{
		ID: 0x5C,
	}
	d1 := protocol.NewMockDevice(config1)
	d2 := protocol.NewMockDevice(config2)
	d3 := protocol.NewMockDevice(config3)
	c := protocol.NewDeviceChain(d1, d2, d3)
	h := protocol.NewHandler(c, protocol.NoLogging)

	brDesc := []protocol.BulkReadDescriptor{
		{
			ID:     byte(config1.ID),
			Addr:   1,
			Length: 4,
		},
		{
			ID:     byte(config2.ID),
			Addr:   4,
			Length: 10,
		},
		{
			ID:     byte(config3.ID),
			Addr:   2,
			Length: 22,
		},
	}
	got, err := h.BulkRead(brDesc)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if len(got) != len(brDesc) {
		t.Fatalf("Expected all %d devices to return status, only %d did", len(brDesc), len(got))
	}
	for i, v := range got {
		if len(v) != int(brDesc[i].Length) {
			t.Errorf("Expected %d bytes from response %d, got %d", int(brDesc[i].Length), i+1, len(v))
		}
	}
}

func TestBulkWrite(t *testing.T) {
	config1 := protocol.MockDeviceConfig{
		ID: 0x5A,
	}
	config2 := protocol.MockDeviceConfig{
		ID: 0x5B,
	}
	config3 := protocol.MockDeviceConfig{
		ID: 0x5C,
	}
	d1 := protocol.NewMockDevice(config1)
	d2 := protocol.NewMockDevice(config2)
	d3 := protocol.NewMockDevice(config3)
	c := protocol.NewDeviceChain(d1, d2, d3)
	h := protocol.NewHandler(c, protocol.NoLogging)

	bwDesc := []protocol.BulkWriteDescriptor{
		{
			ID:   byte(config1.ID),
			Addr: 2,
			Data: []byte{0x01, 0x02, 0x03},
		},
		{
			ID:   byte(config2.ID),
			Addr: 4,
			Data: []byte{0x04, 0x05},
		},
		{
			ID:   byte(config3.ID),
			Addr: 5,
			Data: []byte{0x06},
		},
	}
	if err := h.BulkWrite(bwDesc); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}
