package protocol

import (
	"fmt"
	"io"
)

const (
	NoLogging byte = 0
	LogRead   byte = 1 << iota
	LogWrite
	LogReadWrite = LogRead | LogWrite
)

type PacketLogger struct {
	rw        io.ReadWriter
	logWriter io.Writer
	level     byte
}

func NewPacketLogger(readWriter io.ReadWriter, level byte, destination io.Writer) *PacketLogger {
	return &PacketLogger{
		rw:        readWriter,
		logWriter: destination,
		level:     level,
	}
}

func (l *PacketLogger) Read(p []byte) (n int, err error) {
	n, err = l.rw.Read(p)
	if l.level&LogRead != 0 && err == nil {
		N := n
		if N > len(p) {
			N = len(p)
		}
		fmt.Fprintf(l.logWriter, "PacketLoger: Read %d bytes: %v\n", n, p[:N])
	}
	return n, err
}

func (l *PacketLogger) Write(p []byte) (n int, err error) {
	n, err = l.rw.Write(p)
	if l.level&LogWrite != 0 && err == nil {
		fmt.Fprintf(l.logWriter, "PacketLoger: Wrote %d bytes: %v\n", n, p)
	}
	return n, err
}
