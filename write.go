package simplelog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

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

func NewWrite(filename string, maxSize int64) (*Write, error) {
	w := Write{
		currentFilename:   filename,
		singleFileMaxSize: maxSize,
		queue:             make(chan []byte, 100000),
		exit:              make(chan struct{}, 1),
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

func (w *Write) Append(message []byte) error {
	return w.AppendCtx(context.Background(), message)
}

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

func (w *Write) Close() error {
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

func (w *Write) loop() {
	for {
		select {
		case <-w.exit:
			return
		case msg := <-w.queue:
			n, err := w.file.Write(msg)
			if err != nil {
				err = w.Append(msg)
				if err != nil {
					fmt.Println("w.file.Write", err)
				}
			}
			w.isRoll(w.moreThan(n))
		}
	}
}

func (w *Write) moreThan(size int) bool {
	return atomic.AddInt64(&w.currentFileSize, int64(size)) > w.singleFileMaxSize
}

func (w *Write) isRoll(roll bool) {
	if roll {
		err := w.file.Close()
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
