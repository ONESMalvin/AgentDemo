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

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
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
		WatchChild(host)
	}

	//fmt.Println("Total:", v.Total, " Free:", v.Free, " Used:", v.Used, " UsedPercent:", v.UsedPercent)
	//BornAChild()
}

type PluginHost struct {
	ExecCmd *exec.Cmd
	LogFile *os.File
}

func WatchChild(childHost *PluginHost) {
	fmt.Println(fmt.Sprintf("Let me wait for the child[%d] to finish", childHost.ExecCmd.Process.Pid))
	c := make(chan bool, 1)
	// 等待子进程完成
	/*
		go func() {
			err := childHost.ExecCmd.Wait()
			if err != nil {
				fmt.Println("Error waiting for child process:", err)
			}
			c <- true
		}()
	*/

	defer childHost.LogFile.WriteString(fmt.Sprintf("【Agent】:Child[%d] has finished\n", childHost.ExecCmd.Process.Pid))
	for {
		select {
		case <-c:
			return
		case <-time.After(3 * time.Second):
			rss, vsz, shm, err := utils.GetProcessMemoryUsage(childHost.ExecCmd.Process.Pid)
			if err == nil {
				childHost.LogFile.WriteString(fmt.Sprintf("【Agent】：Child[%d] RSS:%s VSZ:%s SHM:%s\n",
					childHost.ExecCmd.Process.Pid, utils.ByteToKb(uint64(rss)), utils.ByteToKb(uint64(vsz)), utils.ByteToKb(uint64(shm))))
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
	host.ExecCmd = cmd
	host.LogFile = f
	return host, nil
}
