package simplelog

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestSimpleLogger_Debug(t *testing.T) {
	sl, err := NewSimpleLogger(DefaultConfig())
	if err != nil {
		t.Fatal(err)
		return
	}
	count := 10
	for count > 0 {
		count--
		sl.Debug("%s_%d", "TestSimpleLogger_Debug", count)
	}
	sl.Close()
}

func TestSimpleLogger_Info(t *testing.T) {
	sl, err := NewSimpleLogger(DefaultConfig())
	if err != nil {
		t.Fatal(err)
		return
	}
	count := 10
	for count > 0 {
		count--
		sl.Info("%s_%d", "TestSimpleLogger_Info", count)
	}
	sl.Close()
}

func TestSimpleLogger_Warn(t *testing.T) {
	sl, err := NewSimpleLogger(DefaultConfig())
	if err != nil {
		t.Fatal(err)
		return
	}
	count := 10
	for count > 0 {
		count--
		sl.Warn("%s_%d", "TestSimpleLogger_Warn", count)
	}
	sl.Close()
}

func TestSimpleLogger_Error(t *testing.T) {
	sl, err := NewSimpleLogger(DefaultConfig())
	if err != nil {
		t.Fatal(err)
		return
	}
	count := 10
	for count > 0 {
		count--
		sl.Error("%s_%d", "TestSimpleLogger_Error", count)
	}
	sl.Close()
}

func TestSimpleLogger_Stack(t *testing.T) {
	sl, err := NewSimpleLogger(DefaultConfig())
	if err != nil {
		t.Fatal(err)
		return
	}

	sl.Error("%s_%s", "TestSimpleLogger_Fatal", sl.Stack(fmt.Errorf("这是stack")))

	sl.Close()
}

func TestSimpleLogger_Size(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Compress = true
	cfg.MaxSize = 1024 * 1024
	sl, err := NewSimpleLogger(cfg)
	if err != nil {
		t.Fatal(err)
		return
	}
	count := 100
	for count > 0 {
		count--
		sl.Info("TestSimpleLogger_Size")
		sl.Debug("TestSimpleLogger_Size")
		sl.Warn("TestSimpleLogger_Size")
		sl.Error("TestSimpleLogger_Size")
	}

	sl.Close()
}

func TestSimpleLogger_Keep(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAge = 10 * time.Second
	sl, err := NewSimpleLogger(cfg)
	if err != nil {
		t.Fatal(err)
		return
	}

	sl.Info("info")

	sl.Close()
}

func TestSimpleLogger_SetLevel(t *testing.T) {
	sl, err := NewSimpleLogger(DefaultConfig())
	if err != nil {
		t.Fatal(err)
		return
	}

	sl.SetLevel(ErrorLevel)

	sl.Debug("TestSimpleLogger_SetLevel")

	sl.Close()

	content, _ := ioutil.ReadFile(sl.cfg.Filename)
	if strings.Index(string(content), "TestSimpleLogger_SetLevel") != -1 {
		t.Fail()
	}

}
