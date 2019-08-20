package main

import (
	"fmt"
	"github.com/huzhongqing/simplelog"
)

func main(){
	cfg := simplelog.DefaultConfig()
	slog, err := simplelog.NewSimpleLogger(cfg)
	if err != nil {
		// some thing
		panic(err)
		return
	}

	slog.Info("example %s", "demo")

	slog.Error("example %s", "error")

	slog.Fatal("example %s", slog.Stack(fmt.Errorf("stack info")))

	slog.Close()
}
