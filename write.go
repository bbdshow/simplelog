package simplelog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

// Write 写入文件对象
type Write struct {
	// 当前文件名
	currentFilename string
	// 当前文件容量
	currentFileSize int64
	// 单文件最大容量 bytes
	singleFileMaxSize int64

	queue chan []byte

	isExit int32
	exit   chan struct{}
	// 文件对象
	file *os.File
}

// NewWrite new
func NewWrite(filename string, maxSize int64) (*Write, error) {
	w := Write{
		currentFilename:   filename,
		singleFileMaxSize: maxSize,
		queue:             make(chan []byte, 10000),
		exit:              make(chan struct{}, 1),
	}

	dir := path.Dir(filename)
	err := os.MkdirAll(dir, os.ModePerm|os.ModeDir)
	if err != nil {
		return nil, err
	}

	// 查看文件信息
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsExist(err) {
			return nil, err
		} else {
			// 不存在文件
			f, err := os.Create(w.currentFilename)
			if err != nil {
				return nil, err
			}
			w.file = f
		}
	} else {
		// 存在文件
		f, err := os.OpenFile(w.currentFilename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return nil, err
		}
		w.file = f
		w.currentFileSize = info.Size()
	}

	// 写文件
	go w.loop()

	return &w, nil
}

// AppendCtx 追加写入文件
func (w *Write) AppendCtx(ctx context.Context, message []byte) error {
	if atomic.LoadInt32(&w.isExit) == 1 {
		return errors.New("write closed")
	}

	select {
	case w.queue <- message:
	case <-ctx.Done():
		return errors.New("ctx done")
	}
	return nil
}

// Close 释放
func (w *Write) Close() error {
	time.Sleep(10 * time.Millisecond)
	atomic.StoreInt32(&w.isExit, 1)
	for {
		if len(w.queue) == 0 {
			w.exit <- struct{}{}
			close(w.queue)
			close(w.exit)
			return w.file.Close()
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// loop 持续写入文件
func (w *Write) loop() {
	for {
		select {
		case <-w.exit:
			return
		case msg := <-w.queue:
			n, err := w.file.Write(msg)
			if err != nil {
				err = w.AppendCtx(context.Background(), msg)
				if err != nil {
					fmt.Println("w.file.Write", err, msg)
				}
			}
			w.isRoll(w.moreThan(n))
		}
	}
}

// moreThan 容量是否超过
func (w *Write) moreThan(size int) bool {
	return atomic.AddInt64(&w.currentFileSize, int64(size)) > w.singleFileMaxSize
}

// isRoll 是否滚动文件
func (w *Write) isRoll(roll bool) {
	if roll {
		var err error
		err = w.file.Close()
		if err != nil {
			fmt.Printf("simple log file close %s \n", err.Error())
			return
		} else {
			backupFilename := strings.Replace(w.currentFilename, ".log", fmt.Sprintf(".%s.log", w.timeSuffix()), 1)
			err = os.Rename(w.currentFilename, backupFilename)
			if err != nil {
				fmt.Printf("simple log file rename %s \n", err.Error())
				return
			} else {
			create:
				f, err := os.Create(w.currentFilename)
				if err != nil {
					fmt.Printf("simple log file create %s \n", err.Error())
					time.Sleep(10 * time.Millisecond)
					goto create
				}
				w.currentFileSize = 0
				w.file = f
			}
		}
	}

}

func (w *Write) timeSuffix() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05.00000")
}
