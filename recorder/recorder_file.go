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

// FileRecorder is responsible for writing records to a file. It can filter events
// to only show certain types, or limit them to certain IP/Bitcoin addresses.
// It will periodically rotate the files and supports compression.
type FileRecorder struct {
	wg         *sync.WaitGroup
	cmdConfig  map[string]bool
	ipConfig   map[string]bool
	addrConfig map[string]bool
	fileTimer  *time.Timer
	sigWriter  chan struct{}
	txtQ       chan string
	txIndex    *parmap.ParMap
	blockIndex *parmap.ParMap

	log  adaptor.Log
	comp adaptor.Compressor

	filePath string
	fileName string

	fileSize int64
	fileAge  time.Duration

	file *os.File

	done uint32
}

// New creates a new recorder with the given options.
func NewFileRecorder(options ...func(*FileRecorder)) (*FileRecorder, error) {
	rec := &FileRecorder{
		wg:         &sync.WaitGroup{},
		cmdConfig:  make(map[string]bool),
		ipConfig:   make(map[string]bool),
		addrConfig: make(map[string]bool),
		sigWriter:  make(chan struct{}),
		txtQ:       make(chan string, 1),
		txIndex:    parmap.New(),
		blockIndex: parmap.New(),

		filePath: "logs/",
		fileName: time.Now().String(),
		fileSize: 1 * 1024 * 1024,
		fileAge:  1 * 60 * time.Minute,
	}

	for _, option := range options {
		option(rec)
	}

	if rec.comp == nil {
		rec.comp = compressor.NewLZ4()
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
func SetLog(log adaptor.Log) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		rec.log = log
	}
}

// SetCompressor injects the compression wrapper to be used on rotation.
func SetCompressor(comp adaptor.Compressor) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		rec.comp = comp
	}
}

// SetFilePath sets the directory path to the files into.
func SetFilePath(path string) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		rec.filePath = path
	}
}

// SetSizeLimit sets the size limit upon which the logs will rotate.
func SetSizeLimit(size int64) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		rec.fileSize = size
	}
}

// SetAgeLimit sets the file age upon which the logs will rotate.
func SetAgeLimit(age time.Duration) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		rec.fileAge = age
	}
}

// SetTypes sets the type of events to write to file.
func FilterTypes(cmds ...string) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		for _, cmd := range cmds {
			rec.cmdConfig[cmd] = true
		}
	}
}

func FilterIPs(ips ...string) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		for _, ip := range ips {
			rec.ipConfig[ip] = true
		}
	}
}

func FilterAddresses(addrs ...string) func(*FileRecorder) {
	return func(rec *FileRecorder) {
		for _, addr := range addrs {
			rec.addrConfig[addr] = true
		}
	}
}

// Message will process a given message and log it if it's elligible.
func (rec *FileRecorder) Message(msg wire.Message, ra *net.TCPAddr,
	la *net.TCPAddr) {
	if len(rec.cmdConfig) > 0 {
		if !rec.cmdConfig[msg.Command()] {
			return
		}
	}

	if len(rec.ipConfig) > 0 {
		if !rec.ipConfig[ra.IP.String()] {
			return
		}
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
		tx := NewTransactionRecord(m, ra, la)
		ok := true

		if len(rec.addrConfig) > 0 {
			ok = false
		Outer:
			for _, out := range tx.details.outs {
				for _, addr := range out.addrs {
					if rec.addrConfig[addr.EncodeAddress()] {
						ok = true
						break Outer
					}
				}
			}
		}

		if !ok {
			return
		}

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
func (rec *FileRecorder) Stop() {
	if atomic.SwapUint32(&rec.done, 1) == 1 {
		return
	}

	close(rec.sigWriter)

	rec.wg.Wait()
}

func (rec *FileRecorder) startup() {
	rec.wg.Add(1)
	go rec.goWriter()
}

func (rec *FileRecorder) goWriter() {
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

func (rec *FileRecorder) checkTime() {
	if rec.fileAge == 0 {
		return
	}

	rec.rotateLog()

	rec.fileTimer.Reset(rec.fileAge)
}

func (rec *FileRecorder) checkSize() {
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

func (rec *FileRecorder) rotateLog() {
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

func (rec *FileRecorder) compressLog() {
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
