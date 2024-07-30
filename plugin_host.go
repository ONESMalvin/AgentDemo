package main

import (
	"context"
	"fmt"
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

	TerminateFunc context.CancelFunc
}

func StartHost() (*PluginHost, error) {
	host := &PluginHost{}

	ctx := context.Background()
	cmdCtx, cancelFunc := context.WithCancel(ctx)
	host.TerminateFunc = cancelFunc
	cmd := exec.CommandContext(cmdCtx, "child/child")
	//cmd := exec.Command("child/child")
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	// kill child process when parent process dead
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	cmd.WaitDelay = 5 * time.Second
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
			fmt.Println("Error waiting for child process:", err)
		}
		host.childExit <- true
	}()
	host.ExecCmd = cmd
	host.LogFile = f
	host.Available = true
	return host, nil
}

func (p *PluginHost) TerminateHost() {
	if p.Available == false {
		return
	}
	p.LogFile.WriteString(fmt.Sprintf("【Agent】:I'm going to kill the child process[%d]\n", p.ExecCmd.Process.Pid))
	if p.ExecCmd == nil || p.ExecCmd.Process == nil {
		log.Println("[Warning]No process to kill")
	}
	p.TerminateFunc()
	p.Available = false
}
