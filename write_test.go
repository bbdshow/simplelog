package simplelog

import (
	"context"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	dir = "/Users/hzq/workspace/go_modules/github.com/simplelog/testdata"
)

func TestNewWrite(t *testing.T) {
	_, err := NewWrite(dir+"/TestNewWrite.log", 100)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrite_Append(t *testing.T) {
	w, err := NewWrite(dir+"/TestWrite_Append.log", 100)
	if err != nil {
		t.Fatal(err)
	}

	err = w.Append([]byte("append"))
	if err != nil {
		t.Fatal(err)
	}

	w.Close()

	w1, err := NewWrite(dir+"/TestWrite_Append.log", 100)
	if err != nil {
		t.Fatal(err)
	}

	err = w1.Append([]byte("append2"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrite_AppendCtx(t *testing.T) {
	w, err := NewWrite(dir+"/TestWrite_AppendCtx.log", 100)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	err = w.AppendCtx(ctx, []byte("append"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrite_isRoll(t *testing.T) {
	w, err := NewWrite(dir+"/TestWrite_isRoll.log", 50)
	if err != nil {
		t.Fatal(err)
	}
	count := 30
	for count > 0 {
		count--
		w.Append([]byte("size test" + strconv.Itoa(count)))
	}

	time.Sleep(50 * time.Millisecond)

	err = w.Close()
	if err != nil {
		t.Log(err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
		return
	}

	for _, v := range files {
		if strings.Index(v.Name(), "TestWrite_isRoll.2") != -1 {
			t.Log("backup", v.Name())
			return
		}
	}
	t.Fatal("backup error")
}
