package simplelog

import (
	"archive/tar"
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

const (
	_compressSuffix = ".tar.gz"
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

	compress    bool
	compressing chan struct{} // 正在压缩

	// 文件对象
	file *os.File
}

type WriteConfig struct {
	// 写入文件
	Filename string
	// 单文件最大 bytes
	MaxSize int64
	// 保留文件时间
	MaxAge time.Duration
	// Gzip 压缩
	Compress bool
}

func DefaultWriteConfig() *WriteConfig {
	return &WriteConfig{
		Filename: "./log/logger.log",
		MaxSize:  1024 * 1024 * 100,
		MaxAge:   time.Duration(7*24) * time.Hour,
		Compress: true,
	}
}

// NewWrite new
func NewWrite(cfg *WriteConfig) *Write {
	if cfg == nil {
		cfg = DefaultWriteConfig()
	}

	w := Write{
		currentDir:        path.Dir(cfg.Filename),
		currentFilename:   cfg.Filename,
		singleFileMaxSize: cfg.MaxSize,
		maxAge:            cfg.MaxAge,
		compress:          cfg.Compress,
		compressing:       make(chan struct{}, 1),
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
	tick := time.NewTicker(30 * time.Second)
	for {
		w.mutex.Lock()
		if w.file != nil {
			err = w.file.Close()
		}
		w.mutex.Unlock()

		select {
		case w.compressing <- struct{}{}:
			goto exit
		case <-tick.C:
			goto exit
		}
	}
exit:
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
				go w.compressFile(backupFilename)
			}

			// 切割的时候检查一下文件时间
			go w.maxAgeFile()
		}
	}
}

func (w *Write) timeSuffix() string {
	now := time.Now()
	return now.Format("2006-01-02T15:04:05.00000")
}

// 压缩文件
func (w *Write) compressFile(filename string) {
	select {
	case w.compressing <- struct{}{}:
		defer func() {
			<-w.compressing
		}()

		if filename == "" {
			return
		}

		destName := fmt.Sprintf("%s%s", filename, _compressSuffix)
		dest, err := os.Create(destName)
		if err != nil {
			fmt.Printf("failed to open compressed log file: %v \n", err)
			break
		}
		defer dest.Close()

		gz := gzip.NewWriter(dest)
		defer gz.Close()

		tw := tar.NewWriter(gz)
		defer tw.Close()

		file, err := os.Open(filename)
		if err != nil {
			fmt.Printf("failed to open log file %s : %v \n", filename, err)
			break
		}
		defer file.Close()
		destInfo, _ := file.Stat()

		tHeader, err := tar.FileInfoHeader(destInfo, destInfo.Name())
		if err != nil {
			break
		}
		_ = tw.WriteHeader(tHeader)

		if _, err := io.Copy(tw, file); err != nil {
			fmt.Printf("log tar gzip %v \n", err)
			break
		}

		if err := os.Remove(filename); err != nil {
			fmt.Printf("remove source file: %v \n", err)
		}
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
