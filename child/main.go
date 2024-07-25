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
	log.Printf("New child pid:%d", os.Getpid())
	utils.AsyncAllocBuffer(2*1024*1024, 10*time.Second, true)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	for {
		select {
		case sig := <-c:
			if sig == syscall.SIGTERM {
				log.Printf("Child[%d] process received SIGTERM signal, ignore...\n", os.Getpid())
			} else {
				log.Println("capture signal: ", sig.String())
				log.Fatalln("Shutdown Server ...")
			}
		}
	}
}
