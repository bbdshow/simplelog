package simplelog

import "testing"

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
}
