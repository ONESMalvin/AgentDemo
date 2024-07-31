package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ONESMalvin/agent-demo/utils"
)

const HEARTBEAT_PIPE = 3

func main() {
	log.Printf("New child pid:%d", os.Getpid())
	utils.AsyncAllocBuffer(2*1024*1024, 10*time.Second, true)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	ticker := time.NewTicker(1 * time.Second)
	hbFile := os.NewFile(uintptr(HEARTBEAT_PIPE), "/tmp/heartbeat")
	for {
		select {
		case <-ticker.C:
			hbFile.Write([]byte("heartbeat"))
		case sig := <-c:
			if sig == syscall.SIGTERM {
				//log.Printf("Child[%d] process received SIGTERM signal, ignore...\n", os.Getpid())
				log.Printf("Child[%d] process received SIGTERM signal, exit...\n", os.Getpid())
				os.Exit(0)
			} else {
				log.Println("capture signal: ", sig.String())
				log.Fatalln("Shutdown Server ...")
			}
		}
	}
}
