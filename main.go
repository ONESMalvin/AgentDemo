package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/ONESMalvin/agent-demo/utils"
)

var mode int64

func main() {
	cmd, err := BornAChild()
	if err == nil {
		WatchChild(cmd)
	}

	//fmt.Println("Total:", v.Total, " Free:", v.Free, " Used:", v.Used, " UsedPercent:", v.UsedPercent)
	//BornAChild()
}

func WatchChild(childCmd *exec.Cmd) {
	fmt.Println(fmt.Sprintf("Let me wait for the child[%d] to finish", childCmd.Process.Pid))
	c := make(chan bool, 1)
	// 等待子进程完成
	go func() {
		err := childCmd.Wait()
		if err != nil {
			fmt.Println("Error waiting for child process:", err)
		}
		c <- true
	}()
	defer fmt.Println(fmt.Sprintf("Child[%d] has finished", childCmd.Process.Pid))
	for {
		select {
		case <-c:
			return
		case <-time.After(3 * time.Second):
			rss, vsz, err := utils.GetProcessMemoryUsage(childCmd.Process.Pid)
			if err == nil {
				fmt.Println(fmt.Sprintf("Child[%d] RSS:%s VSZ:%s", childCmd.Process.Pid, utils.ByteToKb(uint64(rss)), utils.ByteToKb(uint64(vsz))))
			}
		}
	}
}

func BornAChild() (*exec.Cmd, error) {
	path, err := exec.LookPath("child/child")
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(path)
	cmd := exec.Command("child/child")
	cmd.Stdin = nil
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err.Error())
	}
	f, err := os.OpenFile("child.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}
	go io.Copy(f, stdout)
	go io.Copy(f, stderr)

	err = cmd.Start()
	if err != nil {
		fmt.Println("Failed to start child process:", err)
		return nil, err
	}
	return cmd, nil
}
