package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type PluginHost struct {
	ExecCmd   *exec.Cmd
	LogFile   *os.File
	childExit chan bool
	Available bool

	terminateFunc context.CancelFunc

	HearBeatPipe           *os.File
	LastHeartbeatTimeStamp int64
}

func StartHost() (*PluginHost, error) {
	host := &PluginHost{}

	ctx := context.Background()
	cmdCtx, cancelFunc := context.WithCancel(ctx)
	host.terminateFunc = cancelFunc
	cmd := exec.CommandContext(cmdCtx, "child/child")
	//cmd := exec.Command("child/child")
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	// kill child process when parent process dead
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	cmd.WaitDelay = 5 * time.Second

	hbr, hbw, err := os.Pipe()
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	cmd.ExtraFiles = append(cmd.ExtraFiles, hbw)
	defer hbw.Close()
	host.HearBeatPipe = hbr
	// close stdin
	cmd.Stdin = nil
	// redirect Stdout & Stderr
	f, err := os.OpenFile("child.log", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}
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
	host.childExit = make(chan bool)
	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("【Agent】Child process[%d] exit with error:%s\n", cmd.Process.Pid, err.Error())
		}
		host.childExit <- true
	}()
	go host.CheckHeartbeat()
	host.ExecCmd = cmd
	host.LogFile = f
	host.Available = true
	return host, nil
}

func (p *PluginHost) TerminateHost() {
	if p.Available == false {
		return
	}
	if p.ExecCmd != nil && p.ExecCmd.Process != nil && p.ExecCmd.ProcessState == nil {
		p.LogFile.WriteString(fmt.Sprintf("【Agent】:I'm going to kill the child process[%d]\n", p.ExecCmd.Process.Pid))
		p.terminateFunc()
	}
	p.Available = false
}

func (p *PluginHost) CleanUp() {
	fmt.Printf("【Agent】:Child[%d] clean up\n", p.ExecCmd.Process.Pid)
	p.HearBeatPipe.Close()
	p.LogFile.Close()
}

func (p *PluginHost) CheckHeartbeat() {
	for {
		select {
		case <-time.After(3 * time.Second):
			if p.Available == false {
				return
			}
			buffer := make([]byte, 512)
			n, err := p.HearBeatPipe.Read(buffer)
			if err != nil && err != io.EOF {
				if p.Available {
					log.Printf("【Agent】:Child[%d] read heartbeat error:%s\n", p.ExecCmd.Process.Pid, err.Error())
				}
			}
			if n > 0 {
				p.LastHeartbeatTimeStamp = time.Now().Unix()
				log.Printf("【Agent】:Child[%d] heartbeat, len(%d)\n", p.ExecCmd.Process.Pid, n)
			}
		}
	}
}
