package recorder

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"

	"github.com/CIRCL/pbtc/adaptor"
)

type Recorder struct {
	wg        *sync.WaitGroup
	cmdConfig map[string]bool
	fileTimer *time.Timer
	sigWriter chan struct{}
	txtQ      chan string
	binQ      chan []byte

	log adaptor.Logger

	filePath string
	fileName string

	fileSize int64
	fileAge  time.Duration

	txtFile *os.File
	binFile *os.File

	done uint32
}

func New(options ...func(*Recorder)) (*Recorder, error) {
	rec := &Recorder{
		wg:        &sync.WaitGroup{},
		cmdConfig: make(map[string]bool),
		sigWriter: make(chan struct{}, 1),
		txtQ:      make(chan string, 1),
		binQ:      make(chan []byte, 1),

		filePath: "records/",
		fileName: time.Now().String(),
		fileSize: 1,
		fileAge:  1 * time.Minute,
	}

	for _, option := range options {
		option(rec)
	}

	txtFile, err := os.Create(rec.filePath + rec.fileName + ".txt")
	if err != nil {
		return nil, err
	}

	binFile, err := os.Create(rec.filePath + rec.fileName + ".bin")
	if err != nil {
		txtFile.Close()
		return nil, err
	}

	rec.txtFile = txtFile
	rec.binFile = binFile
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

func (rec *Recorder) Message(msg wire.Message) {
	if !rec.cmdConfig[msg.Command()] {
		return
	}

	switch m := msg.(type) {
	case *wire.MsgVersion:
		rec.recordVersion(m)

	case *wire.MsgAddr:
		rec.recordAddr(m)

	case *wire.MsgInv:
		rec.recordInv(m)

	case *wire.MsgTx:
		rec.recordTx(m)
	}
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

		/*case <-rec.fileTimer.C:
		rec.rotateLogs()*/

		case txt := <-rec.txtQ:
			_, err := rec.txtFile.WriteString(txt + "\n")
			if err != nil {
				rec.log.Error("[REC] Could not write txt file (%v)", err)
			}

			stat, _ := rec.txtFile.Stat()
			if stat.Size() >= rec.fileSize*1024 {
				rec.rotateLogs()
			}

		case bin := <-rec.binQ:
			_, err := rec.binFile.Write(bin)
			if err != nil {
				rec.log.Error("[REC] Could not write bin file (%v)", err)
			}

			stat, _ := rec.binFile.Stat()
			if stat.Size() >= rec.fileSize*1024 {
				rec.rotateLogs()
			}
		}
	}
}

func (rec *Recorder) rotateLogs() {
	_ = rec.binFile.Close()
	_ = rec.txtFile.Close()

	rec.fileName = time.Now().String()

	txtFile, err := os.Create(rec.filePath + rec.fileName + ".txt")
	if err != nil {
		panic(err)
	}

	binFile, err := os.Create(rec.filePath + rec.fileName + ".bin")
	if err != nil {
		txtFile.Close()
		panic(err)
	}

	rec.txtFile = txtFile
	rec.binFile = binFile

	rec.fileTimer.Reset(rec.fileAge)
}

func (rec *Recorder) recordVersion(msg *wire.MsgVersion) {
	record := NewVersionRecord(msg)
	rec.txtQ <- record.String()
	rec.binQ <- record.Bytes()
}

func (rec *Recorder) recordVerAck(msg *wire.MsgVerAck) {

}

func (rec *Recorder) recordAddr(msg *wire.MsgAddr) {
	record := NewAddressRecord(msg)
	rec.txtQ <- record.String()
	rec.binQ <- record.Bytes()
}

func (rec *Recorder) recordInv(msg *wire.MsgInv) {
	record := NewInventoryRecord(msg)
	rec.txtQ <- record.String()
	rec.binQ <- record.Bytes()
}

func (rec *Recorder) recordGetData(msg *wire.MsgGetData) {

}

func (rec *Recorder) recordNotFound(msg *wire.MsgNotFound) {

}

func (rec *Recorder) recordGetBlocks(msg *wire.MsgGetBlocks) {

}

func (rec *Recorder) recordGetHeaders(msg *wire.MsgGetHeaders) {

}

func (rec *Recorder) recordTx(msg *wire.MsgTx) {
	record := NewTransactionRecord(msg)
	rec.txtQ <- record.String()
	rec.binQ <- record.Bytes()
}

func (rec *Recorder) recordBlock(msg *wire.MsgBlock) {

}

func (rec *Recorder) recordHeaders(msg *wire.MsgHeaders) {

}

func (rec *Recorder) recordGetAddr(msg *wire.MsgHeaders) {

}

func (rec *Recorder) recordMemPool(msg *wire.MsgMemPool) {

}

func (rec *Recorder) recordPing(msg *wire.MsgPing) {

}

func (rec *Recorder) recordPong(msg *wire.MsgPong) {

}

func (rec *Recorder) recordReject(msg *wire.MsgReject) {

}

func (rec *Recorder) recordFilterLoad(msg *wire.MsgFilterLoad) {

}

func (rec *Recorder) recordFilterAdd(msg *wire.MsgFilterAdd) {

}

func (rec *Recorder) recordFilterClear(msg *wire.MsgFilterClear) {

}

func (rec *Recorder) recordMerkleBlock(msg *wire.MsgMerkleBlock) {

}

func (rec *Recorder) recordAlert(msg *wire.MsgAlert) {

}
