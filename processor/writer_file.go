package processor

import (
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/compressor"
)

const Version = "PBTC Log Version 1"

type FileWriter struct {
	Processor

	wg        *sync.WaitGroup
	comp      adaptor.Compressor
	filePath  string
	fileSize  int64
	fileAge   time.Duration
	fileTimer *time.Timer
	file      *os.File
	sigWriter chan struct{}
	txtQ      chan string
	done      uint32
}

func NewFileWriter(options ...func(adaptor.Processor)) (*FileWriter, error) {
	w := &FileWriter{
		filePath: "logs/",
		fileSize: 1 * 1024 * 1024,
		fileAge:  1 * 60 * time.Minute,

		sigWriter: make(chan struct{}),
		wg:        &sync.WaitGroup{},
		txtQ:      make(chan string, 1),
	}

	for _, option := range options {
		option(w)
	}

	if w.comp == nil {
		w.comp = compressor.NewDummy()
	}

	_, err := os.Stat(w.filePath)
	if err != nil {
		err := os.MkdirAll(w.filePath, 0777)
		if err != nil {
			return nil, err
		}
	}

	w.rotateLog()

	w.fileTimer = time.NewTimer(w.fileAge)

	w.wg.Add(1)
	go w.goWriter()

	return w, nil
}

// SetCompressor injects the compression wrapper to be used on rotation.
func SetCompressor(comp adaptor.Compressor) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.comp = comp
	}
}

// SetFilePath sets the directory path to the files into.
func SetFilePath(path string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.filePath = path
	}
}

// SetSizeLimit sets the size limit upon which the logs will rotate.
func SetSizeLimit(size int64) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.fileSize = size
	}
}

// SetAgeLimit sets the file age upon which the logs will rotate.
func SetAgeLimit(age time.Duration) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.fileAge = age
	}
}

// Stop ends the execution of the recorder sub-routines and returns once
// everything was stopped cleanly.
func (w *FileWriter) Close() {
	if atomic.SwapUint32(&w.done, 1) == 1 {
		return
	}

	close(w.sigWriter)

	w.wg.Wait()
}

func (w *FileWriter) Process(record adaptor.Record) {
	w.txtQ <- record.String()
}

func (w *FileWriter) goWriter() {
	defer w.wg.Done()

WriteLoop:
	for {
		select {
		case _, ok := <-w.sigWriter:
			if !ok {
				break WriteLoop
			}

		case <-w.fileTimer.C:
			w.checkTime()

		case txt := <-w.txtQ:
			_, err := w.file.WriteString(txt + "\n")
			if err != nil {
				w.log.Error("[REC] Could not write txt file (%v)", err)
			}

			w.checkSize()
		}
	}

	w.file.Close()
}

func (w *FileWriter) checkTime() {
	if w.fileAge == 0 {
		return
	}

	w.rotateLog()

	w.fileTimer.Reset(w.fileAge)
}

func (w *FileWriter) checkSize() {
	if w.fileSize == 0 {
		return
	}

	fileStat, err := w.file.Stat()
	if err != nil {
		panic(err)
	}

	if fileStat.Size() < w.fileSize {
		return
	}

	w.rotateLog()
}

func (w *FileWriter) rotateLog() {
	file, err := os.Create(w.filePath +
		time.Now().Format(time.RFC3339) + ".txt")
	if err != nil {
		w.log.Error("Could not create file (%v)", err)
		return
	}

	_, err = file.WriteString("#" + Version + "\n")
	if err != nil {
		w.log.Error("Could not write to file (%v)", err)
		return
	}

	if w.file != nil {
		w.compressLog()
		err = w.file.Close()
		if err != nil {
			w.log.Warning("[REC] Could not close file on rotate (%v)", err)
		}
	}

	w.file = file
}

func (w *FileWriter) compressLog() {
	_, err := w.file.Seek(0, 0)
	if err != nil {
		w.log.Warning("[REC] Failed to seek output file (%v)", err)
		return
	}

	output, err := os.Create(w.file.Name() + ".out")
	if err != nil {
		w.log.Critical("[REC] Failed to create output file (%v)", err)
		return
	}

	writer, err := w.comp.GetWriter(output)
	if err != nil {
		w.log.Error("[REC] Failed to create output writer (%v)", err)
		return
	}

	_, err = io.Copy(writer, w.file)
	if err != nil {
		w.log.Error("[REC] Failed to compress log file (%v)", err)
		return
	}
}
