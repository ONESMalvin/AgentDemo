package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/ONESMalvin/agent-demo/utils"
)

var mode int64

var hosts []*PluginHost

// start child process
func RegisterSIGUSR1() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)
	go func() {
		for {
			<-sigs
			host, err := BornAChild()
			if err == nil {
				hosts = append(hosts, host)
			}
		}
	}()
	return nil
}

// kill child process
func RegisterSIGUSR2() error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)
	go func() {
		for {
			<-sigs
			deleteHost := hosts[0]
			hosts = hosts[1:]
			deleteHost.LogFile.WriteString(fmt.Sprintf("【Agent】:I'm going to dead, kill the child process[%d]\n", deleteHost.ExecCmd.Process.Pid))
			deleteHost.LogFile.Close()
			deleteHost.ExecCmd.Process.Kill()
		}
	}()
	return nil
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// 等待信号
		<-sigChan
		for _, host := range hosts {
			host.LogFile.WriteString(fmt.Sprintf("【Agent】:I'm going to dead, kill the child process[%d]\n", host.ExecCmd.Process.Pid))
			host.LogFile.Close()
			host.ExecCmd.Process.Kill()
		}
		os.Exit(0)
	}()
	hosts = make([]*PluginHost, 0)
	host, err := BornAChild()
	if err == nil {
		hosts = append(hosts, host)
	}
	WatchChilds(hosts)

	//fmt.Println("Total:", v.Total, " Free:", v.Free, " Used:", v.Used, " UsedPercent:", v.UsedPercent)
	//BornAChild()
}

type PluginHost struct {
	ExecCmd         *exec.Cmd
	LogFile         *os.File
	childExitSignal chan bool
}

func WatchChilds(hosts []*PluginHost) {
	for {
		select {
		case <-time.After(3 * time.Second):
			for _, childHost := range hosts {
				rss, vsz, shm, err := utils.GetProcessMemoryUsage(childHost.ExecCmd.Process.Pid)
				if err == nil {
					childHost.LogFile.WriteString(fmt.Sprintf("【Agent】：Child[%d] RSS:%s VSZ:%s SHM:%s\n",
						childHost.ExecCmd.Process.Pid, utils.ByteToKb(uint64(rss)), utils.ByteToKb(uint64(vsz)), utils.ByteToKb(uint64(shm))))
				}
			}
		default:
			for _, childHost := range hosts {
				select {
				case <-childHost.childExitSignal:
					childHost.LogFile.WriteString(fmt.Sprintf("【Agent】:Child[%d] has finished\n", childHost.ExecCmd.Process.Pid))
				default:
				}
			}
		}
	}
}

func BornAChild() (*PluginHost, error) {
	host := &PluginHost{}
	f, err := os.OpenFile("child.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}
	cmd := exec.Command("child/child")
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGHUP}
	cmd.Stdin = nil
	//cmd.Stdout = os.Stdout
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
	c := make(chan bool, 1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for child process:", err)
		}
		c <- true
	}()
	host.ExecCmd = cmd
	host.LogFile = f
	host.childExitSignal = c
	return host, nil
}
