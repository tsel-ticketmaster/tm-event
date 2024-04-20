package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tsel-ticketmaster/tm-event/pkg/applogger"
)

func main() {
	logger := applogger.GetLogrus()
	go func() {
		for i := 0; i < 100; i++ {
			logger.Infof("Task A: %d", i)

			time.Sleep(time.Millisecond * 1750)
		}
		logger.Infof("Task A: Done")

	}()
	go func() {
		for i := 0; i < 100; i++ {
			logger.Infof("Task B: %d", i)

			time.Sleep(time.Millisecond * 3500)
		}
		logger.Infof("Task B: Done")
	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGINT)
	<-sigterm

}
