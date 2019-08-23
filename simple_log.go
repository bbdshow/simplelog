package simplelog

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*

	保留日志天数，定时扫描 dir
*/

// SimpleLogger
type SimpleLogger struct {
	level  int32
	cfg    Config
	format *Format

	rwMutex  sync.RWMutex
	writeMap map[string]*Write // key = level value=write

	exit chan struct{}
}

const (
	DebugLevel int = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// Config 日志配置信息
type Config struct {
	// 写入文件夹
	Dir string
	// 单文件最大 bytes
	MaxSize int64
	// 保留文件时间
	KeepTime time.Duration

	// 短方法名
	ShortFuncname int
	// 调用深度
	Calldpeth int
	// 日志等级，>= 设置的等级才会写入
	Level int
	// 格式化，配置
	Format FormatConfig
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		Dir:           "./log",
		MaxSize:       1024 * 1024 * 10, // 10mb
		KeepTime:      time.Duration(7*24) * time.Hour,
		Calldpeth:     3,
		ShortFuncname: 2,
		Level:         DebugLevel,
		Format:        DefaultFormatConfig(),
	}
}

// NewSimpleLogger new
func NewSimpleLogger(cfg Config) (*SimpleLogger, error) {
	sl := SimpleLogger{
		level:    int32(cfg.Level),
		cfg:      cfg,
		format:   NewFormat(cfg.Format),
		exit:     make(chan struct{}, 1),
		writeMap: make(map[string]*Write, 5),
	}

	go sl.checkKeepTime(sl.exit)

	return &sl, nil
}

// Debug
func (sl *SimpleLogger) Debug(format string, args ...interface{}) {
	if !sl.isWrite(DebugLevel) {
		return
	}
	sl.Write(context.Background(), true, "DEBUG", fmt.Sprintf(format, args...))
}

// Info
func (sl *SimpleLogger) Info(format string, args ...interface{}) {
	if !sl.isWrite(InfoLevel) {
		return
	}
	sl.Write(context.Background(), true, "INFO", fmt.Sprintf(format, args...))
}

// Warn
func (sl *SimpleLogger) Warn(format string, args ...interface{}) {
	if !sl.isWrite(WarnLevel) {
		return
	}
	sl.Write(context.Background(), true, "WARN", fmt.Sprintf(format, args...))
}

// Error
func (sl *SimpleLogger) Error(format string, args ...interface{}) {
	if !sl.isWrite(ErrorLevel) {
		return
	}
	sl.Write(context.Background(), true, "ERROR", fmt.Sprintf(format, args...))
}

// Fatal 等级
func (sl *SimpleLogger) Fatal(format string, args ...interface{}) {
	if !sl.isWrite(FatalLevel) {
		return
	}
	sl.Write(context.Background(), true, "FATAL", fmt.Sprintf(format, args...))
}

func (sl *SimpleLogger) WarpFlag(message string) string {
	return fmt.Sprintf(sl.format.GetFormatFlag(), message)
}

// String 生成待写入文件的数据
func (sl *SimpleLogger) String(call bool, level, message string) string {
	msg := ""
	if call {
		_, file, line, ok := runtime.Caller(sl.cfg.Calldpeth)
		if !ok {
			file = "???"
			line = 0
		}
		if sl.cfg.ShortFuncname > 0 {
			temp := strings.Split(file, "/")
			if len(temp) >= sl.cfg.ShortFuncname {
				temp = temp[len(temp)-sl.cfg.ShortFuncname:]
			}
			file = strings.Join(temp, "/")
		}
		msg = sl.format.GenMessage(level, file+":"+strconv.Itoa(line)+" "+message)
	} else {
		msg = sl.format.GenMessage(level, message)
	}

	return msg
}

// Write 写入文件
func (sl *SimpleLogger) Write(ctx context.Context, call bool, level, message string) error {
	write, err := sl.loadOrCreateWrite(level)
	if err != nil {
		return err
	}
	return write.AppendCtx(ctx, []byte(sl.String(call, level, message)))
}

// SetLevel 设置错误等级
func (sl *SimpleLogger) SetLevel(level int) {
	atomic.StoreInt32(&sl.level, int32(level))
}

// Stack 堆栈信息
func (sl *SimpleLogger) Stack(err error) string {
	return sl.format.Stack(err.Error())
}

// Close write 存在缓存队列，一定要 close 保证队列的所有数据都写入磁盘
func (sl *SimpleLogger) Close() error {
	sl.rwMutex.RLock()
	defer sl.rwMutex.RUnlock()

	for k, v := range sl.writeMap {
		err := v.Close()
		if err != nil {
			fmt.Printf("close %s %s\n", k, err.Error())
		}
	}

	sl.exit <- struct{}{}

	return nil
}

// checkKeepTime 检查文件保留时间
func (sl *SimpleLogger) checkKeepTime(exit chan struct{}) {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		// 先执行一把
		files, err := ioutil.ReadDir(sl.cfg.Dir)
		if err == nil {
			for _, file := range files {
				if time.Now().Add(-sl.cfg.KeepTime).After(file.ModTime()) {
					// 删除此文件
					os.Remove(path.Join(sl.cfg.Dir, file.Name()))
				}
			}
		}
		select {
		case <-ticker.C:

		case <-exit:
			ticker.Stop()

			return
		}
	}
}

// loadOrCreateWrite 加载或创建写入对象
func (sl *SimpleLogger) loadOrCreateWrite(level string) (*Write, error) {
	sl.rwMutex.RLock()
	w, ok := sl.writeMap[level]
	sl.rwMutex.RUnlock()

	if !ok {
		w, err := NewWrite(path.Join(sl.cfg.Dir, fmt.Sprintf("%s.log", strings.ToLower(level))), sl.cfg.MaxSize)
		if err != nil {
			return nil, err
		}
		sl.rwMutex.Lock()
		sl.writeMap[level] = w
		sl.rwMutex.Unlock()

		return w, nil
	}

	return w, nil
}

func (sl *SimpleLogger) isWrite(level int) bool {
	return int32(level) >= atomic.LoadInt32(&sl.level)
}

var stdout io.Writer = os.Stderr

// Output 系统输出
func (sl *SimpleLogger) Output(calldepth int, s string) error {
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}

	msg := sl.format.GenMessage("output", file+":"+strconv.Itoa(line)+" "+s)
	_, err := stdout.Write([]byte(msg))
	return err
}
