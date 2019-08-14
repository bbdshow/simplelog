package simplelog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Write struct {
	// 当前文件名
	currentFilename string
	// 当前文件容量
	currentFileSize int64
	// 单文件最大容量
	singleFileMaxSize int64

	queue chan []byte
	// 文件对象
	file *os.File
}

func NewWrite(filename string, maxSize int64) (*Write, error) {
	w := Write{
		currentFilename:   filename,
		singleFileMaxSize: maxSize,
		queue:             make(chan []byte, 100000),
	}

	// 查看文件信息
	info, err := os.Stat(filename)
	if err != nil {
		if !os.IsExist(err) {
			return nil, err
		} else {
			// 不存在文件
			f, err := os.Create(w.currentFilename)
			if err != nil {
				return nil, err
			}
			w.file = f
		}
	}

	// 存在文件
	f, err := os.OpenFile(w.currentFilename, os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	w.file = f
	w.currentFileSize = info.Size()

	return &w, nil
}

func (w *Write) Append(message []byte) error {
	return w.AppendCtx(context.Background(), message)
}

func (w *Write) AppendCtx(ctx context.Context, message []byte) error {
	select {
	case w.queue <- message:
	case <-ctx.Done():
		return errors.New("ctx done")
	}
	return nil
}

func (w *Write) loop() {
	for {
		select {
		case msg := <-w.queue:
			n, err := w.file.Write(msg)
			if err != nil {
				w.queue <- msg
			} else {
				w.isRoll(w.moreThan(n))
			}
		}
	}
}

func (w *Write) moreThan(size int) bool {
	return (w.currentFileSize + int64(size)) > w.singleFileMaxSize
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
	return now.Format("2006-01-02 15:04:05")
}
