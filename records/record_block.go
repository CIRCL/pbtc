package records

import (
	"bytes"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/wire"
)

type BlockRecord struct {
	Record

	hdr     *HeaderRecord
	details []*DetailsRecord
}

func NewBlockRecord(msg *wire.MsgBlock, ra *net.TCPAddr,
	la *net.TCPAddr) *BlockRecord {
	record := &BlockRecord{
		Record: Record{
			stamp: time.Now(),
			ra:    ra,
			la:    la,
			cmd:   msg.Command(),
		},

		hdr:     NewHeaderRecord(&msg.Header),
		details: make([]*DetailsRecord, len(msg.Transactions)),
	}

	for i, tx := range msg.Transactions {
		record.details[i] = NewDetailsRecord(tx)
	}

	return record
}

func (br *BlockRecord) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString(br.stamp.Format(time.RFC3339Nano))
	buf.WriteString(Delimiter1)
	buf.WriteString(br.cmd)
	buf.WriteString(Delimiter1)
	buf.WriteString(br.ra.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(br.la.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(br.hdr.String())
	buf.WriteString(Delimiter1)
	buf.WriteString(strconv.FormatInt(int64(len(br.details)), 10))

	buf.WriteString(Delimiter1)
	for _, tx := range br.details {
		buf.WriteString(Delimiter2)
		buf.WriteString(tx.String())
	}

	return buf.String()
}
