// Copyright (c) 2015 Max Wolter
// Copyright (c) 2015 CIRCL - Computer Incident Response Center Luxembourg
//                           (c/o smile, security made in Lëtzebuerg, Groupement
//                           d'Intérêt Economique)
//
// This file is part of PBTC.
//
// PBTC is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// PBTC is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with PBTC.  If not, see <http://www.gnu.org/licenses/>.

package processor

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/CIRCL/pbtc/adaptor"
	"github.com/CIRCL/pbtc/compressor"
)

const Version = "PBTC Log Version 1"

type FileWriter struct {
	Processor

	wg        *sync.WaitGroup
	comp      adaptor.Compressor
	fileTimer *time.Timer
	file      *os.File
	sig       chan struct{}
	txtQ      chan string

	filePath      string
	filePrefix    string
	fileName      string
	fileSuffix    string
	fileSizelimit int64
	fileAgelimit  time.Duration
}

func NewFileWriter(options ...func(adaptor.Processor)) (*FileWriter, error) {
	w := &FileWriter{
		filePath:      "logs/",
		filePrefix:    "pbtc-",
		fileName:      "2006-01-02T15:04:05Z07:00",
		fileSuffix:    ".log",
		fileSizelimit: 1048576,
		fileAgelimit:  3600 * time.Second,

		sig:  make(chan struct{}),
		wg:   &sync.WaitGroup{},
		txtQ: make(chan string, 1),
	}

	for _, option := range options {
		option(w)
	}

	if w.comp == nil {
		w.comp = compressor.NewDummy()
	}

	err := os.MkdirAll(w.filePath, 0777)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// SetCompressor injects the compression wrapper to be used on rotation.
func SetFileCompressor(comp adaptor.Compressor) func(adaptor.Processor) {
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

func SetFilePrefix(prefix string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.filePrefix = prefix
	}
}

func SetFileName(name string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.fileName = name
	}
}

func SetFileSuffix(suffix string) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.fileSuffix = suffix
	}
}

// SetSizeLimit sets the size limit upon which the logs will rotate.
func SetFileSizelimit(sizelimit int64) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.fileSizelimit = sizelimit
	}
}

// SetAgeLimit sets the file age upon which the logs will rotate.
func SetFileAgelimit(agelimit time.Duration) func(adaptor.Processor) {
	return func(pro adaptor.Processor) {
		w, ok := pro.(*FileWriter)
		if !ok {
			return
		}

		w.fileAgelimit = agelimit
	}
}

func (w *FileWriter) Start() {
	w.log.Info("[WF] Start: begin")

	w.rotateLog()

	w.fileTimer = time.NewTimer(w.fileAgelimit)

	w.wg.Add(1)
	go w.goProcess()

	w.log.Info("[WF] Start: completed")
}

func (w *FileWriter) Stop() {
	w.log.Info("[WF] Stop: begin")

	close(w.sig)
	w.wg.Wait()

	w.log.Info("[WF] Stop: completed")
}

func (w *FileWriter) Process(record adaptor.Record) {
	w.log.Debug("[WF] Process: %v", record.Command())

	w.txtQ <- record.String()
}

func (w *FileWriter) goProcess() {
	defer w.wg.Done()

WriteLoop:
	for {
		select {
		case _, ok := <-w.sig:
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
	if w.fileAgelimit == 0 {
		return
	}

	w.rotateLog()

	w.fileTimer.Reset(w.fileAgelimit)
}

func (w *FileWriter) checkSize() {
	if w.fileSizelimit == 0 {
		return
	}

	fileStat, err := w.file.Stat()
	if err != nil {
		panic(err)
	}

	if fileStat.Size() < w.fileSizelimit {
		return
	}

	w.rotateLog()
}

func (w *FileWriter) rotateLog() {
	stamp := time.Now().Format(w.fileName)
	file, err := os.Create(w.filePath + w.filePrefix + stamp + w.fileSuffix)
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
