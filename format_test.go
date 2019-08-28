package simplelog

import (
	"strings"
	"testing"
)

func TestFormat_Stack(t *testing.T) {
	f := NewFormat(DefaultFormatConfig())
	if strings.Index(f.Stack("-"), "Traceback:") != -1 {
		return
	}
	t.Fatal()
}

func TestFormat_GenMessage(t *testing.T) {
	f := NewFormat(DefaultFormatConfig())
	t.Log(f.GenMessage("DEBUG", "TestFormat_GenMessage"))
}
