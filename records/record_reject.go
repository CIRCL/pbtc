package records

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type RejectRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	cmd    string
	code   uint8
	reject string
	hash   []byte
	reason string
}

func NewRejectRecord(msg *wire.MsgReject, ra *net.TCPAddr,
	la *net.TCPAddr) *RejectRecord {
	record := &RejectRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		cmd:    msg.Command(),
		code:   uint8(msg.Code),
		reject: msg.Cmd,
		hash:   msg.Hash.Bytes(),
		reason: msg.Reason,
	}

	return record
}

func (rr *RejectRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(rr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(rr.code), 10))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.reject)
	buf.WriteString(Delimiter1)
	buf.WriteString(hex.EncodeToString(rr.hash))
	buf.WriteString(Delimiter1)
	buf.WriteString(rr.reason)

	return buf.String()
}

func (rr *RejectRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(rr.cmd))    //  1
	binary.Write(buf, binary.LittleEndian, rr.stamp.UnixNano())     //  8
	binary.Write(buf, binary.LittleEndian, rr.ra.IP.To16())         // 16
	binary.Write(buf, binary.LittleEndian, uint16(rr.ra.Port))      //  2
	binary.Write(buf, binary.LittleEndian, rr.la.IP.To16())         // 16
	binary.Write(buf, binary.LittleEndian, uint16(rr.la.Port))      //  2
	binary.Write(buf, binary.LittleEndian, rr.code)                 //  1
	binary.Write(buf, binary.LittleEndian, ParseCommand(rr.reject)) //  1
	binary.Write(buf, binary.LittleEndian, rr.hash)                 // 32
	binary.Write(buf, binary.LittleEndian, uint32(len(rr.reason)))  //  4
	binary.Write(buf, binary.LittleEndian, rr.reason)               //  X

	// total: 83 + X
	return buf.Bytes()
}
