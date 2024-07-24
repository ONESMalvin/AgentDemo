package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ONESMalvin/agent-demo/utils"
)

func main() {
	utils.AsyncAllocBuffer(2*1024*1024, 10*time.Second, true)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	<-c
	log.Println("Shutdown Server ...")
}
