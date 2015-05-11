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

type GetBlocksRecord struct {
	stamp  time.Time
	ra     *net.TCPAddr
	la     *net.TCPAddr
	cmd    string
	stop   [32]byte
	hashes [][32]byte
}

func NewGetBlocksRecord(msg *wire.MsgGetBlocks, ra *net.TCPAddr,
	la *net.TCPAddr) *GetBlocksRecord {
	record := &GetBlocksRecord{
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

func (gr *GetBlocksRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(gr.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(gr.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(hex.EncodeToString(gr.stop[:]))
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(gr.hashes)), 10))

	for _, hash := range gr.hashes {
		buf.WriteString(Delimiter2)
		buf.WriteString(hex.EncodeToString(hash[:]))
	}

	return buf.String()
}

func (gr *GetBlocksRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, ParseCommand(gr.cmd))   //  1
	binary.Write(buf, binary.LittleEndian, gr.stamp.UnixNano())    //  8
	binary.Write(buf, binary.LittleEndian, gr.ra.IP.To16())        // 16
	binary.Write(buf, binary.LittleEndian, uint16(gr.ra.Port))     //  2
	binary.Write(buf, binary.LittleEndian, gr.la.IP.To16())        // 16
	binary.Write(buf, binary.LittleEndian, uint16(gr.la.Port))     //  2
	binary.Write(buf, binary.LittleEndian, gr.stop)                // 32
	binary.Write(buf, binary.LittleEndian, uint32(len(gr.hashes))) //  4

	for _, hash := range gr.hashes { // N
		binary.Write(buf, binary.LittleEndian, hash) // 32
	}

	// total: 81 + N*32
	return buf.Bytes()
}
