package protocol_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/haguro/go-dxl/protocol/v2"
)

var device1Config = protocol.MockDeviceConfig{
	ID:              0x99,
	ModelNumber:     0x424,
	FirmwareVer:     0x2F,
	ControlTableRAM: []byte{0x32, 0x14, 0xF0, 0xE9, 0xA9, 0x7C},
}

var device2Config = protocol.MockDeviceConfig{
	ID:              0xF0,
	ModelNumber:     0x424,
	FirmwareVer:     0x2F,
	ControlTableRAM: []byte{0xA9, 0x56, 0xFF, 0x93, 0xBB, 0x7C},
}

var device3Config = protocol.MockDeviceConfig{
	ID:              0x1B,
	ModelNumber:     0x424,
	FirmwareVer:     0x2F,
	ControlTableRAM: []byte{0x99, 0xAF, 0x12, 0x98, 0x7D, 0xE3},
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
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)

	got, err := h.Ping(byte(device1Config.ID))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := protocol.PingResponse{
		ID:       byte(device1Config.ID),
		Model:    uint16(device1Config.ModelNumber),
		Firmware: byte(device1Config.FirmwareVer),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestRead(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr, length := 3, 2

	got, err := h.Read(byte(device1Config.ID), uint16(addr), uint16(length))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := device1Config.ControlTableRAM[addr : addr+length]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestWrite(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 2
	data := []byte{0xF1, 0xF2}
	if err := h.Write(byte(device1Config.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	got := d.InspectControlTable(addr, len(data))
	if !reflect.DeepEqual(got, data) {
		t.Errorf("Expected %+v, got %+v", data, got)
	}
}

func TestRegWrite(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 3
	data := []byte{0xF1, 0xF2}
	if err := h.RegWrite(byte(device1Config.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := append([]byte{byte(addr), byte(addr >> 8)}, data...)
	got := d.InspectRegWriteBuffer()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected RegWrite to not change control table, got %+v", got)
	}
}

func TestAction(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	addr := 1
	data := []byte{0xF1, 0xF2}
	if err := h.RegWrite(byte(device1Config.ID), uint16(addr), data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.Action(byte(device1Config.ID)); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	got := d.InspectControlTable(addr, len(data))
	if !reflect.DeepEqual(got, data) {
		t.Errorf("Expected %+v, got %+v", data, got)
	}
}

func TestReboot(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.Reboot(byte(device1Config.ID)); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := make([]byte, len(device1Config.ControlTableRAM))
	got := d.InspectControlTable(0, len(device1Config.ControlTableRAM))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestClear(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	if err := h.Clear(byte(device1Config.ID), protocol.ClearMultiRotationPos); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := []byte{0x32, 0x04, 0, 0}
	got := d.InspectControlTable(0, 4)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestFactoryReset(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	newCTVals := []byte{0xF1, 0xFA, 0x09, 0xA0}
	if err := h.Write(byte(device1Config.ID), 0, newCTVals...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.FactoryReset(byte(device1Config.ID), protocol.ResetExceptID); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	got := d.InspectControlTable(0, len(device1Config.ControlTableRAM))
	if !reflect.DeepEqual(got, device1Config.ControlTableRAM) {
		t.Errorf("Expected %+v, got %+v", device1Config.ControlTableRAM, got)
	}
}

func TestControlTableBackup(t *testing.T) {
	d := protocol.NewMockDevice(device1Config)
	h := protocol.NewHandler(d, protocol.NoLogging)
	newCTVals := []byte{0xF1, 0xFA, 0x09, 0xA0}
	if err := h.ControlTableBackup(byte(device1Config.ID), protocol.BackupStore); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.Write(byte(device1Config.ID), 0, newCTVals...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	if err := h.ControlTableBackup(byte(device1Config.ID), protocol.BackupRestore); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
	got := d.InspectControlTable(0, len(device1Config.ControlTableRAM))
	if !reflect.DeepEqual(got, device1Config.ControlTableRAM) {
		t.Errorf("Expected %+v, got %+v", device1Config.ControlTableRAM, got)
	}
}

func TestSyncRead(t *testing.T) {
	d1 := protocol.NewMockDevice(device1Config)
	d2 := protocol.NewMockDevice(device2Config)
	d3 := protocol.NewMockDevice(device3Config)
	c := protocol.NewDeviceChain(d1, d2, d3)
	h := protocol.NewHandler(c, protocol.NoLogging)

	addr, length := 3, 2
	ids := []byte{byte(device1Config.ID), byte(device2Config.ID), byte(device3Config.ID)}
	got, err := h.SyncRead(ids, uint16(addr), uint16(length))
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	want := [][]byte{
		device1Config.ControlTableRAM[addr : addr+length],
		device2Config.ControlTableRAM[addr : addr+length],
		device3Config.ControlTableRAM[addr : addr+length],
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Expected %+v, got %+v", want, got)
	}
}

func TestSyncWrite(t *testing.T) {
	d1 := protocol.NewMockDevice(device1Config)
	d2 := protocol.NewMockDevice(device2Config)
	d3 := protocol.NewMockDevice(device3Config)
	c := protocol.NewDeviceChain(d1, d2, d3)
	h := protocol.NewHandler(c, protocol.NoLogging)

	addr := 4
	data1 := []byte{0xF1, 0xF2}
	data2 := []byte{0xA7, 0xA8}
	data3 := []byte{0x21, 0x43}
	data := []byte{byte(device1Config.ID)}
	data = append(data, data1...)
	data = append(data, byte(device2Config.ID))
	data = append(data, data2...)
	data = append(data, byte(device3Config.ID))
	data = append(data, data3...)

	if err := h.SyncWrite(uint16(addr), 2, data...); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	got1 := d1.InspectControlTable(addr, len(data1))
	if !reflect.DeepEqual(got1, data1) {
		t.Errorf("Expected %+v to be written to Device 1, got %+v", data1, got1)
	}

	got2 := d2.InspectControlTable(addr, len(data2))
	if !reflect.DeepEqual(got2, data2) {
		t.Errorf("Expected %+v to be written to Device 2, got %+v", data2, got2)
	}

	got3 := d3.InspectControlTable(addr, len(data3))
	if !reflect.DeepEqual(got3, data3) {
		t.Errorf("Expected %+v to be written to Device 3, got %+v", data2, got2)
	}
}

		t.Fatalf("Expected no error, got %q", err)
	}
	got := d.InspectControlTable(0, len(deviceConfig.ControlTableRAM))
	if !reflect.DeepEqual(got, deviceConfig.ControlTableRAM) {
		t.Errorf("Expected %+v, got %+v", deviceConfig.ControlTableRAM, got)
	}
}
