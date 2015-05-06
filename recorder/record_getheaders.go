package recorder

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type GetHeadersRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	cmd    string
	stop   [32]byte
	hashes [][32]byte
}

func NewGetHeadersRecord(msg *wire.MsgGetHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *GetHeadersRecord {
	record := &GetHeadersRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		cmd:    msg.Command(),
		stop:   msg.HashStop,
		hashes: make([][32]byte, len(msg.BlockLocatorHashes)),
	}

	for i, hash := range msg.BlockLocatorHashes {
		record.hashes[i] = *hash
	}

	return record
}

func (gr *GetHeadersRecord) String() string {
	buf := new(bytes.Buffer)

	// line 1: header
	buf.WriteString(gr.cmd)
	buf.WriteString(" ")
	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(gr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(gr.la.String())
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(gr.stop[:]))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(gr.hashes)), 10))

	// line 2: locator hashes
	buf.WriteString("\n")
	for _, hash := range gr.hashes {
		buf.WriteString(" ")
		buf.WriteString(hex.EncodeToString(hash[:]))
	}

	return buf.String()
}

func (gr *GetHeadersRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, gr.stamp.UnixNano())
	binary.Write(buf, binary.LittleEndian, gr.ra.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(gr.ra.Port))
	binary.Write(buf, binary.LittleEndian, gr.la.IP.To16())
	binary.Write(buf, binary.LittleEndian, uint16(gr.la.Port))
	binary.Write(buf, binary.LittleEndian, ParseCommand(gr.cmd))
	binary.Write(buf, binary.LittleEndian, len(gr.hashes))

	for _, hash := range gr.hashes {
		binary.Write(buf, binary.LittleEndian, hash)
	}

	return buf.Bytes()
}
