package recorder

import (
	"bytes"
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
	stop   []byte
	hashes [][]byte
}

func NewGetHeadersRecord(msg *wire.MsgGetHeaders, ra *net.TCPAddr,
	la *net.TCPAddr) *GetHeadersRecord {
	record := &GetHeadersRecord{
		stamp:  time.Now(),
		ra:     ra,
		la:     la,
		cmd:    msg.Command(),
		stop:   msg.HashStop.Bytes(),
		hashes: make([][]byte, len(msg.BlockLocatorHashes)),
	}

	for i, hash := range msg.BlockLocatorHashes {
		record.hashes[i] = hash.Bytes()
	}

	return record
}

func (gr *GetHeadersRecord) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(gr.stamp.String())
	buf.WriteString(" ")
	buf.WriteString(gr.ra.String())
	buf.WriteString(" ")
	buf.WriteString(gr.la.String())
	buf.WriteString(" ")
	buf.WriteString(gr.cmd)
	buf.WriteString(" ")
	buf.WriteString(hex.EncodeToString(gr.stop))
	buf.WriteString(" ")
	buf.WriteString(strconv.FormatInt(int64(len(gr.hashes)), 10))

	for _, hash := range gr.hashes {
		buf.WriteString(" ")
		buf.WriteString(hex.EncodeToString(hash))
	}

	return buf.String()
}
