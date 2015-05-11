package recorder

import (
	"bytes"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/parmap"
)

type Recorder struct {
	wg         *sync.WaitGroup
	cmdConfig  map[string]bool
	ipConfig   map[string]bool
	fileTimer  *time.Timer
	sigWriter  chan struct{}
	txtQ       chan string
	binQ       chan []byte
	txIndex    *parmap.ParMap
	blockIndex *parmap.ParMap

	log adaptor.Logger

	filePath string
	fileName string

	fileSize int64
	fileAge  time.Duration

	txtFile *os.File
	binFile *os.File

	done         uint32
	resetLogging bool
	filterCmd    bool
	filterIP     bool
}

func New(options ...func(*Recorder)) (*Recorder, error) {
	rec := &Recorder{
		wg:         &sync.WaitGroup{},
		cmdConfig:  make(map[string]bool),
		ipConfig:   make(map[string]bool),
		sigWriter:  make(chan struct{}),
		txtQ:       make(chan string, 1),
		binQ:       make(chan []byte, 1),
		txIndex:    parmap.New(),
		blockIndex: parmap.New(),

		filePath: "records/",
		fileName: time.Now().String(),
		fileSize: 1 * 1024 * 1024,
		fileAge:  1 * time.Minute,

		resetLogging: false,
		filterCmd:    false,
		filterIP:     false,
	}

	for _, option := range options {
		option(rec)
	}

	if rec.resetLogging {
		err := os.RemoveAll(rec.filePath)
		if err != nil {
			return nil, err
		}
	}

	_, err := os.Stat(rec.filePath)
	if err != nil {
		err := os.MkdirAll(rec.filePath, 0777)
		if err != nil {
			return nil, err
		}
	}

	rec.rotateTxtLog()
	rec.rotateBinLog()

	rec.fileTimer = time.NewTimer(rec.fileAge)

	rec.startup()

	return rec, nil
}

func SetLogger(log adaptor.Logger) func(*Recorder) {
	return func(rec *Recorder) {
		rec.log = log
	}
}

func SetTypes(cmds ...string) func(*Recorder) {
	return func(rec *Recorder) {
		for _, cmd := range cmds {
			rec.cmdConfig[cmd] = true
		}
	}
}

func SetFilePath(path string) func(*Recorder) {
	return func(rec *Recorder) {
		rec.filePath = path
	}
}

func SetFileSize(size int64) func(*Recorder) {
	return func(rec *Recorder) {
		rec.fileSize = size
	}
}

func SetFileAge(age time.Duration) func(*Recorder) {
	return func(rec *Recorder) {
		rec.fileAge = age
	}
}

func EnableReset() func(*Recorder) {
	return func(rec *Recorder) {
		rec.resetLogging = true
	}
}

func (rec *Recorder) Message(msg wire.Message, ra *net.TCPAddr,
	la *net.TCPAddr) {
	if rec.filterCmd && !rec.cmdConfig[msg.Command()] {
		return
	}

	if rec.filterIP && !rec.ipConfig[ra.IP.String()] {
		return
	}

	var record Record

	switch m := msg.(type) {
	case *wire.MsgAddr:
		record = NewAddressRecord(m, ra, la)

	case *wire.MsgAlert:
		record = NewAlertRecord(m, ra, la)

	case *wire.MsgBlock:
		if rec.blockIndex.Has(m.BlockSha()) {
			return
		}

		rec.blockIndex.Insert(m.BlockSha())
		record = NewBlockRecord(m, ra, la)

	case *wire.MsgHeaders:
		record = NewHeadersRecord(m, ra, la)

	case *wire.MsgInv:
		record = NewInventoryRecord(m, ra, la)

	case *wire.MsgPing:
		record = NewPingRecord(m, ra, la)

	case *wire.MsgPong:
		record = NewPongRecord(m, ra, la)

	case *wire.MsgReject:
		record = NewRejectRecord(m, ra, la)

	case *wire.MsgVersion:
		record = NewVersionRecord(m, ra, la)

	case *wire.MsgTx:
		if rec.txIndex.Has(m.TxSha()) {
			return
		}

		rec.txIndex.Insert(m.TxSha())
		record = NewTransactionRecord(m, ra, la)

	case *wire.MsgFilterAdd:
		record = NewFilterAddRecord(m, ra, la)

	case *wire.MsgFilterClear:
		record = NewFilterClearRecord(m, ra, la)

	case *wire.MsgFilterLoad:
		record = NewFilterLoadRecord(m, ra, la)

	case *wire.MsgGetAddr:
		record = NewGetAddrRecord(m, ra, la)

	case *wire.MsgGetBlocks:
		record = NewGetBlocksRecord(m, ra, la)

	case *wire.MsgGetData:
		record = NewGetDataRecord(m, ra, la)

	case *wire.MsgGetHeaders:
		record = NewGetHeadersRecord(m, ra, la)

	case *wire.MsgMemPool:
		record = NewMemPoolRecord(m, ra, la)

	case *wire.MsgMerkleBlock:
		record = NewMerkleBlockRecord(m, ra, la)

	case *wire.MsgNotFound:
		record = NewNotFoundRecord(m, ra, la)

	case *wire.MsgVerAck:
		record = NewVerAckRecord(m, ra, la)
	}

	rec.txtQ <- record.String()
	rec.binQ <- record.Bytes()
}

func (rec *Recorder) Cleanup() {
	rec.shutdown()
	rec.wg.Wait()
	rec.txtFile.Close()
	rec.binFile.Close()
}

func (rec *Recorder) startup() {
	rec.wg.Add(1)
	go rec.goWriter()
}

func (rec *Recorder) shutdown() {
	if atomic.SwapUint32(&rec.done, 1) == 1 {
		return
	}

	close(rec.sigWriter)
}

func (rec *Recorder) goWriter() {
	defer rec.wg.Done()

WriteLoop:
	for {
		select {
		case _, ok := <-rec.sigWriter:
			if !ok {
				break WriteLoop
			}

		case <-rec.fileTimer.C:
			rec.checkTime()

		case txt := <-rec.txtQ:
			_, err := rec.txtFile.WriteString("\n" + txt)
			if err != nil {
				rec.log.Error("[REC] Could not write txt file (%v)", err)
			}

			rec.checkTxtSize()

		case bin := <-rec.binQ:
			_, err := rec.binFile.Write([]byte("\n"))
			if err != nil {
				rec.log.Error("[REC] Could not write newline (%v)", err)
			}

			bin = bytes.Replace(bin, []byte("\n"), []byte("\t\t"), -1)

			_, err = rec.binFile.Write(bin)
			if err != nil {
				rec.log.Error("[REC] Could not write bin file (%v)", err)
			}

			rec.checkBinSize()
		}
	}
}

func (rec *Recorder) checkTime() {
	if rec.fileAge == 0 {
		return
	}

	rec.rotateTxtLog()
	rec.rotateBinLog()

	rec.fileTimer.Reset(rec.fileAge)
}

func (rec *Recorder) checkTxtSize() {
	if rec.fileSize == 0 {
		return
	}

	statTxt, err := rec.txtFile.Stat()
	if err != nil {
		panic(err)
	}

	if statTxt.Size() >= rec.fileSize {
		rec.rotateTxtLog()
	}
}

func (rec *Recorder) checkBinSize() {
	if rec.fileSize == 0 {
		return
	}

	statBin, err := rec.binFile.Stat()
	if err != nil {
		panic(err)
	}

	if statBin.Size() >= rec.fileSize {
		rec.rotateBinLog()
	}
}

func (rec *Recorder) rotateTxtLog() {
	if rec.txtFile != nil {
		err := rec.txtFile.Close()
		if err != nil {
			panic(err)
		}
	}

	txtFile, err := os.Create(rec.filePath +
		time.Now().Format(time.RFC3339) + ".txt")
	if err != nil {
		panic(err)
	}

	txtFile.WriteString("#")
	txtFile.WriteString(Version)
	txtFile.WriteString("\n")

	rec.txtFile = txtFile
}

func (rec *Recorder) rotateBinLog() {
	if rec.binFile != nil {
		err := rec.binFile.Close()
		if err != nil {
			panic(err)
		}
	}

	binFile, err := os.Create(rec.filePath +
		time.Now().Format(time.RFC3339) + ".bin")
	if err != nil {
		panic(err)
	}

	rec.binFile = binFile
}
