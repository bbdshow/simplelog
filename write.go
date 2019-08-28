package simplelog

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Write 写入文件对象
type Write struct {
	mutex sync.Mutex

	// 当前文件夹
	currentDir string

	// 当前文件名
	currentFilename string
	// 当前文件容量
	currentFileSize int64
	// 单文件最大容量 bytes
	singleFileMaxSize int64

	maxAge time.Duration // 0 永久保存

	compress     bool
	compressing  bool // 正在压缩
	compressChan chan string

	// 文件对象
	file *os.File
}

// NewWrite new
func NewWrite(filename string, maxSize int64, maxAge time.Duration, compress bool) *Write {
	w := Write{
		currentDir:        path.Dir(filename),
		currentFilename:   filename,
		singleFileMaxSize: maxSize,
		maxAge:            maxAge,
		compress:          compress,
		compressChan:      make(chan string, 5),
	}

	// 异步压缩文件
	if compress {
		go w.compressFile()
	}

	return &w
}

func (w *Write) Write(message []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.file == nil {
		err = w.openFile()
		if err != nil {
			fmt.Printf("failed create file: %v \n", err)
			return n, err
		}
	}

	n, err = w.file.Write(message)
	if err != nil {
		return n, err
	}
	w.isRoll(w.moreThan(n))

	return n, nil
}

func (w *Write) Sync() error {
	return nil
}

// Close 释放
func (w *Write) Close() error {
	// 等待压缩完成
	var err error
	count := 1000
	for count > 0 {
		count--

		w.mutex.Lock()
		if w.file != nil {
			err = w.file.Close()
		}
		w.mutex.Unlock()

		if !w.compressing {
			close(w.compressChan)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return err
}

func (w *Write) openFile() error {
	dir := path.Dir(w.currentFilename)
	err := os.MkdirAll(dir, os.ModePerm|os.ModeDir)
	if err != nil {
		return err
	}

	// 查看文件信息
	info, err := os.Stat(w.currentFilename)
	if err != nil {
		if os.IsExist(err) {
			return err
		} else {
			// 不存在文件
			f, err := os.Create(w.currentFilename)
			if err != nil {
				return err
			}
			w.file = f
		}
	} else {
		// 存在文件
		f, err := os.OpenFile(w.currentFilename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		w.file = f
		w.currentFileSize = info.Size()
	}

	return nil
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

			// gzip 压缩
			if w.compress {
				w.compressChan <- backupFilename
			}

			// 切割的时候检查一下文件时间
			go w.maxAgeFile()
		}
	}
}

func (w *Write) timeSuffix() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05.00000")
}

// 压缩文件
func (w *Write) compressFile() {
	for {
		select {
		case filename := <-w.compressChan:
			if filename == "" {
				return
			}
			// 正在压缩
			w.compressing = true
			f, err := os.Open(filename)
			if err != nil {
				fmt.Printf("failed to open log file %s : %v \n", filename, err)
				break
			}

			dst := fmt.Sprintf("%s.gzip", filename)
			gzipF, err := os.Create(dst)
			if err != nil {
				fmt.Printf("failed to open compressed log file: %v \n", err)
				break
			}

			gz := gzip.NewWriter(gzipF)

			if _, err := io.Copy(gz, f); err != nil {
				fmt.Printf("log gzip io copy: %v \n", err)
				break
			}

			gz.Close()

			gzipF.Close()

			f.Close()

			if err := os.Remove(filename); err != nil {
				fmt.Printf("failed remove source log file: %v \n", err)
			}
		}

		w.compressing = false
	}
}

func (w *Write) maxAgeFile() {
	if w.maxAge > 0 {
		before := time.Now().Add(-w.maxAge)
		// 扫描当前文件夹里面的日志信息，删除过期的
		// 先执行一把
		files, err := ioutil.ReadDir(w.currentDir)
		if err == nil {
			for _, file := range files {
				// 过期
				if file.ModTime().Before(before) {
					// 删除此文件
					os.Remove(path.Join(w.currentDir, file.Name()))
				}
			}
		}
	}
}
