package recorder

import (
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/compressor"
	"github.com/CIRCL/pbtc/parmap"
)

// Recorder is responsible for writing records to a file. It can filter events
// to only show certain types, or limit them to certain IP/Bitcoin addresses.
// It will periodically rotate the files and supports compression.
type Recorder struct {
	wg         *sync.WaitGroup
	cmdConfig  map[string]bool
	ipConfig   map[string]bool
	fileTimer  *time.Timer
	sigWriter  chan struct{}
	txtQ       chan string
	txIndex    *parmap.ParMap
	blockIndex *parmap.ParMap

	log  adaptor.Logger
	comp adaptor.Compressor

	filePath string
	fileName string

	fileSize int64
	fileAge  time.Duration

	file *os.File

	done         uint32
	resetLogging bool
}

// New creates a new recorder with the given options.
func New(options ...func(*Recorder)) (*Recorder, error) {
	rec := &Recorder{
		wg:         &sync.WaitGroup{},
		cmdConfig:  make(map[string]bool),
		ipConfig:   make(map[string]bool),
		sigWriter:  make(chan struct{}),
		txtQ:       make(chan string, 1),
		txIndex:    parmap.New(),
		blockIndex: parmap.New(),

		filePath: "logs/",
		fileName: time.Now().String(),
		fileSize: 1 * 1024 * 1024,
		fileAge:  1 * 60 * time.Minute,

		resetLogging: false,
	}

	for _, option := range options {
		option(rec)
	}

	if rec.comp == nil {
		rec.comp = compressor.NewLZ4()
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

	rec.rotateLog()

	rec.fileTimer = time.NewTimer(rec.fileAge)

	rec.startup()

	return rec, nil
}

// SetLogger injects the logger to be used for logging.
func SetLogger(log adaptor.Logger) func(*Recorder) {
	return func(rec *Recorder) {
		rec.log = log
	}
}

// SetCompressor injects the compression wrapper to be used on rotation.
func SetCompressor(comp adaptor.Compressor) func(*Recorder) {
	return func(rec *Recorder) {
		rec.comp = comp
	}
}

// SetTypes sets the type of events to write to file.
func SetTypes(cmds ...string) func(*Recorder) {
	return func(rec *Recorder) {
		for _, cmd := range cmds {
			rec.cmdConfig[cmd] = true
		}
	}
}

// SetFilePath sets the directory path to the files into.
func SetFilePath(path string) func(*Recorder) {
	return func(rec *Recorder) {
		rec.filePath = path
	}
}

// SetSizeLimit sets the size limit upon which the logs will rotate.
func SetSizeLimit(size int64) func(*Recorder) {
	return func(rec *Recorder) {
		rec.fileSize = size
	}
}

// SetAgeLimit sets the file age upon which the logs will rotate.
func SetAgeLimit(age time.Duration) func(*Recorder) {
	return func(rec *Recorder) {
		rec.fileAge = age
	}
}

// EnableReset will make the logger delete all previous logs in the given path.
func EnableReset() func(*Recorder) {
	return func(rec *Recorder) {
		rec.resetLogging = true
	}
}

// Message will process a given message and log it if it's elligible.
func (rec *Recorder) Message(msg wire.Message, ra *net.TCPAddr,
	la *net.TCPAddr) {
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
}

// Stop ends the execution of the recorder sub-routines and returns once
// everything was stopped cleanly.
func (rec *Recorder) Stop() {
	if atomic.SwapUint32(&rec.done, 1) == 1 {
		return
	}

	close(rec.sigWriter)

	rec.wg.Wait()
}

func (rec *Recorder) startup() {
	rec.wg.Add(1)
	go rec.goWriter()
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
			_, err := rec.file.WriteString(txt + "\n")
			if err != nil {
				rec.log.Error("[REC] Could not write txt file (%v)", err)
			}

			rec.checkSize()
		}
	}

	rec.file.Close()
}

func (rec *Recorder) checkTime() {
	if rec.fileAge == 0 {
		return
	}

	rec.rotateLog()

	rec.fileTimer.Reset(rec.fileAge)
}

func (rec *Recorder) checkSize() {
	if rec.fileSize == 0 {
		return
	}

	fileStat, err := rec.file.Stat()
	if err != nil {
		panic(err)
	}

	if fileStat.Size() < rec.fileSize {
		return
	}

	rec.rotateLog()
}

func (rec *Recorder) rotateLog() {
	file, err := os.Create(rec.filePath +
		time.Now().Format(time.RFC3339) + ".txt")
	if err != nil {
		return
	}

	_, err = file.WriteString("#" + Version + "\n")
	if err != nil {
		return
	}

	if rec.file != nil {
		rec.compressLog()
		err = rec.file.Close()
		if err != nil {
			rec.log.Warning("[REC] Could not close file on rotate (%v)", err)
		}
	}

	rec.file = file
}

func (rec *Recorder) compressLog() {
	_, err := rec.file.Seek(0, 0)
	if err != nil {
		rec.log.Warning("[REC] Failed to seek output file (%v)", err)
		return
	}

	output, err := os.Create(rec.file.Name() + ".out")
	if err != nil {
		rec.log.Critical("[REC] Failed to create output file (%v)", err)
		return
	}

	writer, err := rec.comp.GetWriter(output)
	if err != nil {
		rec.log.Error("[REC] Failed to create output writer (%v)", err)
		return
	}

	_, err = io.Copy(writer, rec.file)
	if err != nil {
		rec.log.Error("[REC] Failed to compress log file (%v)", err)
		return
	}
}
