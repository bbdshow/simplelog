package simplelog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
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

	// 写入
	write *Write
}

const (
	DebugLevel int = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel

	// 短方法名长度
	_shortFuncname = 2
)

// Config 日志配置信息
type Config struct {
	// 写入文件
	Filename string
	// 单文件最大 bytes
	MaxSize int64
	// 保留文件时间
	MaxAge time.Duration
	// Gzip 压缩
	Compress bool
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
		Filename:  "./log/log.log",
		MaxSize:   1024 * 1024 * 10,   // 10mb
		MaxAge:    7 * 24 * time.Hour, // 7天
		Compress:  false,
		Calldpeth: 3,
		Level:     DebugLevel,
		Format:    DefaultFormatConfig(),
	}
}

// NewSimpleLogger new
func NewSimpleLogger(cfg Config) (*SimpleLogger, error) {
	sl := SimpleLogger{
		level:  int32(cfg.Level),
		cfg:    cfg,
		format: NewFormat(cfg.Format),
	}
	return &sl, nil
}

// Debug
func (sl *SimpleLogger) Debug(format string, args ...interface{}) {
	if !sl.isWrite(DebugLevel) {
		return
	}
	sl.Append(sl.String(sl.cfg.Calldpeth, "DEBUG", fmt.Sprintf(format, args...)))
}

// Info
func (sl *SimpleLogger) Info(format string, args ...interface{}) {
	if !sl.isWrite(InfoLevel) {
		return
	}
	sl.Append(sl.String(sl.cfg.Calldpeth, "INFO", fmt.Sprintf(format, args...)))
}

// Warn
func (sl *SimpleLogger) Warn(format string, args ...interface{}) {
	if !sl.isWrite(WarnLevel) {
		return
	}
	sl.Append(sl.String(sl.cfg.Calldpeth, "WARN", fmt.Sprintf(format, args...)))
}

// Error
func (sl *SimpleLogger) Error(format string, args ...interface{}) {
	if !sl.isWrite(ErrorLevel) {
		return
	}
	sl.Append(sl.String(sl.cfg.Calldpeth, "ERROR", fmt.Sprintf(format, args...)))
}

// Fatal 等级 退出程序
func (sl *SimpleLogger) Fatal(format string, args ...interface{}) {
	if !sl.isWrite(FatalLevel) {
		return
	}
	sl.Append(sl.String(sl.cfg.Calldpeth, "FATAL", fmt.Sprintf(format, args...)))
	os.Exit(-1)
}

// String 生成待写入文件的数据
func (sl *SimpleLogger) String(calldpeth int, level, message string) []byte {

	if calldpeth > 0 {
		_, file, line, ok := runtime.Caller(calldpeth)
		if !ok {
			file = "???"
			line = 0
		}

		temp := strings.Split(file, "/")
		if len(temp) >= _shortFuncname {
			temp = temp[len(temp)-_shortFuncname:]
		}
		file = strings.Join(temp, "/")

		return sl.format.GenMessage(level, file+":"+strconv.Itoa(line)+"\t"+message)
	}

	return sl.format.GenMessage(level, message)
}

// Append 写入文件
func (sl *SimpleLogger) Append(message []byte) (n int, err error) {
	if sl.write == nil {
		sl.write = NewWrite(sl.cfg.Filename, sl.cfg.MaxSize, sl.cfg.MaxAge, sl.cfg.Compress)
	}
	return sl.write.Write(message)
}

// SetLevel 设置错误等级
func (sl *SimpleLogger) SetLevel(level int) {
	atomic.StoreInt32(&sl.level, int32(level))
}

// Stack 堆栈信息
func (sl *SimpleLogger) Stack(err error) string {
	return sl.format.Stack(err.Error())
}

// Close
func (sl *SimpleLogger) Close() error {
	if sl.write != nil {
		return sl.write.Close()
	}
	return nil
}

func (sl *SimpleLogger) isWrite(level int) bool {
	return int32(level) >= atomic.LoadInt32(&sl.level)
}

var stdout io.Writer = os.Stderr

// Output 系统输出
func (sl *SimpleLogger) Output(calldepth int, s string) (n int, err error) {
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	return stdout.Write(sl.format.GenMessage("output", file+":"+strconv.Itoa(line)+" "+s))
}
