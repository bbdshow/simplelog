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

type FormatConfig struct {
	// 时间格式
	LogTimeFormat string
	// 错误等级格式
	LevelFormat string
	// 消息前缀
	MessagePrefix string
}

func NewFormat(cfg FormatConfig) *Format {
	f := Format{
		cfg: cfg,
	}
	return &f
}

func DefaultFormatConfig() FormatConfig {
	return FormatConfig{
		LogTimeFormat: "2006-01-02 15:04:05.000000",
		LevelFormat:   "[%s]",
		MessagePrefix: "",
	}
}

func (f *Format) GenMessage(level, message string) string {
	t := time.Now().Format(f.cfg.LogTimeFormat)
	return fmt.Sprintf("%s %s %s%s\n", t, fmt.Sprintf(f.cfg.LevelFormat, level), f.cfg.MessagePrefix, message)
}

func (f *Format) Error(err error) string {
	if err == nil {
		return fmt.Sprint(err)
	}

	return f.Stack(err.Error())
}

func (f *Format) Stack(msg string) string {
	// runtime 堆栈
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
