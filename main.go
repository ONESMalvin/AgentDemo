package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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
						host.KillHost()
						break
					}
				}
			} else if s == syscall.SIGUSR1 {
				host, err := BornAChild()
				if err == nil {
					hosts = append(hosts, host)
				}
			}
		}
	}
}

type PluginHost struct {
	ExecCmd         *exec.Cmd
	LogFile         *os.File
	childExitSignal chan bool
	Available       bool
}

func (p *PluginHost) KillHost() {
	if p.Available == false {
		return
	}
	p.LogFile.WriteString(fmt.Sprintf("【Agent】:I'm going to dead, kill the child process[%d]\n", p.ExecCmd.Process.Pid))
	if p.ExecCmd == nil || p.ExecCmd.Process == nil {
		log.Println("No process to kill")
		return
	}
	err := p.ExecCmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Printf("Error killing process: %v", err)
		return
	}
	p.ExecCmd.WaitDelay = 3 * time.Second
	p.ExecCmd.Cancel = func() error {
		p.ExecCmd.Process.Kill()
		p.LogFile.WriteString(fmt.Sprintf("【Agent】child process must die[%d]\n", p.ExecCmd.Process.Pid))
		return nil
	}
	err = p.ExecCmd.Wait()
	if err != nil {
		log.Printf("Error waiting for process to exit: %v", err)
	} else {
		log.Printf("Process with PID %d exited successfully", p.ExecCmd.Process.Pid)
	}
	p.Available = false
}

func WatchChilds() {
	go func() {
		for _, childHost := range hosts {
			select {
			case <-childHost.childExitSignal:
				childHost.LogFile.WriteString(fmt.Sprintf("【Agent】:Child[%d] has finished\n", childHost.ExecCmd.Process.Pid))
				childHost.Available = false
			default:
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

func BornAChild() (*PluginHost, error) {
	host := &PluginHost{}
	f, err := os.OpenFile("child.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}
	cmd := exec.Command("child/child")
	// kill child process when parent process dead
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	// close stdin
	cmd.Stdin = nil
	// redirect Stdout & Stderr
	cmd.Stdout = f
	cmd.Stderr = f
	/*
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println(err.Error())
		}
		//cmd.Stderr = os.Stderr
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Println(err.Error())
		}
		go io.Copy(f, stdout)
		go io.Copy(f, stderr)
	*/
	err = cmd.Start()
	if err != nil {
		fmt.Println("Failed to start child process:", err)
		return nil, err
	}
	host.childExitSignal = make(chan bool, 1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for child process:", err)
		}
		host.childExitSignal <- true
	}()
	host.ExecCmd = cmd
	host.LogFile = f
	host.Available = true
	return host, nil
}
