package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type PingRecord struct {
	stamp time.Time
	ra    *net.TCPAddr
	la    *net.TCPAddr
	cmd   string
	nonce uint64
}

func NewPingRecord(msg *wire.MsgPing, ra *net.TCPAddr,
	la *net.TCPAddr) *PingRecord {
	record := &PingRecord{
		stamp: time.Now(),
		ra:    ra,
		la:    la,
		cmd:   msg.Command(),
		nonce: msg.Nonce,
	}

	return record
}

func (pr *PingRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(pr.cmd)
	buf.WriteString(" ")
	buf.WriteString(pr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(" ")
	buf.WriteString(pr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(pr.la.String())
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatUint(pr.nonce, 10))

	return buf.String()
}

func (pr *PingRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(pr.cmd)) // 1
	binary.Write(buf, binary.LittleEndian, pr.stamp.UnixNano())  // 8
	binary.Write(buf, binary.LittleEndian, pr.ra.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(pr.ra.Port))   // 2
	binary.Write(buf, binary.LittleEndian, pr.la.IP.To16())      // 16
	binary.Write(buf, binary.LittleEndian, uint16(pr.la.Port))   // 2
	binary.Write(buf, binary.LittleEndian, pr.nonce)             // 8

	// total: 53
	return buf.Bytes()
}
