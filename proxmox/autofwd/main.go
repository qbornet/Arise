package main

import (
	"autofwd/src/logger"
	"autofwd/src/run"
	"log"
	"os"
	"syscall"

	"github.com/sevlyar/go-daemon"
)

func main() {
	ctx := &daemon.Context{
		PidFileName: "/var/run/autofwd-daemon.pid",
		PidFilePerm: 0o644,
		LogFileName: "/var/log/autofwd-daemon.log",
		LogFilePerm: 0o640,
		Umask:       022,
	}
	if err := logger.InitWithFile(nil, true); err != nil {
		log.Fatalln("error with logger init with file")
	}
	if len(os.Args) != 2 {
		logger.Fatalf("usage: autofwd-daemon [start|stop|restart]")
	}
	switch os.Args[1] {
	case "start":
		d, err := ctx.Reborn()
		if err != nil {
			logger.Fatalf("error when starting daemon: %s", err)
		}
		if d != nil {
			logger.Printf("Daemon start... PID=%d", d.Pid)
			return
		}
		defer ctx.Release()
		logger.Printf("args[1]: %s\n", os.Args[1])

		// Start daemon
		logger.Daemonf("Starting autoforward daemon")
		run.Start()

	case "stop":
		d, err := ctx.Search()
		if err != nil {
			logger.Fatalf("couldn't find daemon: %s", err)
		}
		defer d.Release()
		if err := d.Signal(syscall.SIGTERM); err != nil {
			logger.Fatalf("Sending signaled failed: %s", err)
		}
		if err := os.Remove(ctx.WorkDir + "/" + ctx.PidFileName); err != nil {
			logger.Fatalf("Couldn't remove .pid file: %s", err)
		}
	default:
		logger.Fatalf("usage: autofwd-daemon [start|stop|restart]")
	}
}
