package simplelog

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFormat_Error(t *testing.T) {
	f := NewFormat(DefaultFormatConfig())

	_, err := os.Stat("./not_dir")

	if strings.Index(f.Error(err), "./not_dir: no such file or directory") == -1 {
		t.Fatal()
	}

	t.Log(f.Error(nil))
}

func TestFormat_Stack(t *testing.T) {
	f := NewFormat(DefaultFormatConfig())
	if strings.Index(f.Stack("-"), "Traceback:") != -1 {
		return
	}
	t.Fatal()
}

func TestFormat_GenMessage(t *testing.T) {
	f := NewFormat(DefaultFormatConfig())
	t.Log(f.GenMessage("DEBUG", "TestFormat_GenMessage"+f.Error(fmt.Errorf("xxx"))))
}
