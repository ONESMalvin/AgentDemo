package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ONESMalvin/agent-demo/utils"
)

var mode int64

var hosts []*PluginHost

var mutex sync.Mutex

func main() {
	log.Printf("Agent pid: %d", os.Getpid())
	hosts = make([]*PluginHost, 0)
	WatchChilds()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2, syscall.SIGUSR1)
	// 等待信号
	for {
		select {
		case s := <-sigChan:
			if s == syscall.SIGINT || s == syscall.SIGTERM {
				return
			} else if s == syscall.SIGUSR2 {
				for _, host := range hosts {
					if host.Available {
						host.TerminateHost()
						break
					}
				}
			} else if s == syscall.SIGUSR1 {
				host, err := StartHost()
				if err == nil {
					hosts = append(hosts, host)
				}
			}
		}
	}
}

func WatchChilds() {
	go func() {
		for {
			for _, childHost := range hosts {
				select {
				case <-childHost.childExit:
					childHost.LogFile.WriteString(fmt.Sprintf("【Agent】:Child[%d] has finished\n", childHost.ExecCmd.Process.Pid))
				default:
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-time.After(3 * time.Second):
				for _, childHost := range hosts {
					if !childHost.Available {
						continue
					}
					rss, vsz, shm, err := utils.GetProcessMemoryUsage(childHost.ExecCmd.Process.Pid)
					if err == nil {
						childHost.LogFile.WriteString(fmt.Sprintf("【Agent】：Child[%d] RSS:%s VSZ:%s SHM:%s\n",
							childHost.ExecCmd.Process.Pid, utils.ByteToKb(uint64(rss)), utils.ByteToKb(uint64(vsz)), utils.ByteToKb(uint64(shm))))
					}
				}
			}
		}
	}()
}
