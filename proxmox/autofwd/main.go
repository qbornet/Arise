package main

import (
	"autofwd/src/logger"
	"log"
	"os"

	"github.com/sevlyar/go-daemon"
)

func main() {
	ctx := &daemon.Context{
		PidFileName: "autofwd.pid",
		PidFilePerm: 0644,
		LogFileName: "autofwd.log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
	}
	if err := logger.InitWithFile(nil, true); err != nil {
		log.Fatalln("error with logger init with file")
	}
	if len(os.Args) != 2 {
		logger.Fatalf("usage: autofwd-daemon [start|stop|restart]")
	}
	_ = ctx
	logger.Printf("args[1]: %s\n", os.Args[1])
}
