package simplelog

import (
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	dir      = "./testdata"
	maxAge   = time.Hour
	compress = true
)

func TestWrite_isRoll(t *testing.T) {

	w := NewWrite(&WriteConfig{
		Filename: dir + "/TestWrite_isRoll.log",
		MaxSize:  50,
		MaxAge:   maxAge,
		Compress: compress,
	})

	count := 30
	for count > 0 {
		count--
		w.Write([]byte("size test" + strconv.Itoa(count)))
	}

	time.Sleep(500 * time.Millisecond)

	err := w.Close()
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
