package simplelog

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

/*
	支持一定程度的自定义
	支持输出 error 堆栈信息
*/

type Format struct {
	cfg FormatConfig
}

// FormatConfig
type FormatConfig struct {
	// 时间格式
	LogTimeFormat string

	// 消息前缀
	MessagePrefix string
}

// NewFormat
func NewFormat(cfg FormatConfig) *Format {
	f := Format{
		cfg: cfg,
	}
	return &f
}

// 默认格式化配置文件
func DefaultFormatConfig() FormatConfig {
	return FormatConfig{
		LogTimeFormat: "2006-01-02 15:04:05.000000",
		MessagePrefix: "",
	}
}

// GenMessage 生成等待写入的内容
func (f *Format) GenMessage(level, message string) []byte {
	t := time.Now().Format(f.cfg.LogTimeFormat)

	buf := _BufferPool.Get()
	defer _BufferPool.Put(buf)

	buf.AppendString(t)
	buf.AppendString("\t")
	buf.AppendString("[")
	buf.AppendString(level)
	buf.AppendString("]")
	buf.AppendString("\t")

	if f.cfg.MessagePrefix != "" {
		buf.AppendString(f.cfg.MessagePrefix)
	}

	buf.AppendString(message)

	return buf.Bytes()
}

// 返回当前堆栈信息
func (f *Format) Stack(msg string) string {
	// runtime 堆栈
	//var b bytes.Buffer
	var b strings.Builder
	b.WriteString(msg)
	b.WriteString("\n")
	b.WriteString("Traceback:")
	for _, pc := range *callers() {
		fn := runtime.FuncForPC(pc)
		b.WriteString("\n")
		f, n := fn.FileLine(pc)
		b.WriteString(fmt.Sprintf("%s:%d", f, n))
	}
	//fmt.Println(b.String())
	return b.String()
}

type stack []uintptr

func callers() *stack {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
