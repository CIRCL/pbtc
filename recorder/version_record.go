package recorder

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/util"
)

type VersionRecord struct {
	stamp    time.Time
	version  int32
	services uint64
	rstamp   time.Time
	remote   *net.TCPAddr
	local    *net.TCPAddr
	nonce    uint64
	agent    string
	block    int32
	relay    bool
}

func NewVersionRecord(msg *wire.MsgVersion) *VersionRecord {
	record := &VersionRecord{
		stamp:    time.Now(),
		version:  msg.ProtocolVersion,
		services: uint64(msg.Services),
		rstamp:   msg.Timestamp,
		remote:   util.ParseNetAddress(&msg.AddrYou),
		local:    util.ParseNetAddress(&msg.AddrMe),
		nonce:    msg.Nonce,
		agent:    msg.UserAgent,
		block:    msg.LastBlock,
		relay:    !msg.DisableRelayTx,
	}

	return record
}

func (record *VersionRecord) String() string {
	stamp := record.stamp.String()
	version := strconv.FormatInt(int64(record.version), 10)
	services := strconv.FormatUint(uint64(record.services), 10)
	rstamp := record.rstamp.String()
	remote := record.remote.String()
	local := record.local.String()
	nonce := strconv.FormatUint(record.nonce, 10)
	agent := record.agent
	block := strconv.FormatInt(int64(record.block), 10)
	relay := strconv.FormatBool(record.relay)

	row := strings.Join([]string{stamp, remote, local, rstamp, version, services,
		nonce, agent, block, relay}, " ")

	return row
}

func (record *VersionRecord) Bytes() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, record.stamp)
	binary.Write(buf, binary.LittleEndian, record.version)
	binary.Write(buf, binary.LittleEndian, record.services)
	binary.Write(buf, binary.LittleEndian, record.rstamp)
	binary.Write(buf, binary.LittleEndian, record.remote.IP)
	binary.Write(buf, binary.LittleEndian, int16(record.remote.Port))
	binary.Write(buf, binary.LittleEndian, record.local.IP)
	binary.Write(buf, binary.LittleEndian, int16(record.local.Port))
	binary.Write(buf, binary.LittleEndian, record.nonce)
	binary.Write(buf, binary.LittleEndian, record.agent)
	binary.Write(buf, binary.LittleEndian, record.block)
	binary.Write(buf, binary.LittleEndian, record.relay)

	return buf.Bytes()
}
