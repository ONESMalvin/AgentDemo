package utils

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// AsyncAllocBuffer(2 * 1024, 3 * time.Second, true)
func AsyncAllocBuffer(bufferSize int, timegap time.Duration, showMem bool) {
	fmt.Printf("I'm going to allocate memory asynchronously:%s per %s\n", ByteToKb(uint64(bufferSize)), timegap)
	go func() {
		buf := bytes.Repeat([]byte("1"), bufferSize)
		for {
			select {
			case <-time.After(timegap):
				buf = append(buf, bytes.Repeat([]byte("1"), bufferSize)...)
				if showMem {
					ShowMem()
				}
			}
		}
	}()
}

func GetProcessMemoryUsage(pid int) (rss, vsz, shm int64, err error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0, 0, 0, err
	}
	// default 4KB
	var pageSize int64 = int64(os.Getpagesize())
	var fields []int64
	for _, str := range strings.Split(string(data), " ") {
		v, _ := strconv.ParseInt(str, 10, 64)
		fields = append(fields, v)
	}
	rss = fields[1] * pageSize
	vsz = fields[0] * pageSize
	shm = fields[2] * pageSize
	return rss, vsz, shm, nil
}

func ByteToMb(b uint64) string {
	return fmt.Sprintf("%.3f MB", float64(b)/1024/1024)
}

func ByteToKb(b uint64) string {
	return fmt.Sprintf("%.3f KB", float64(b)/1024)
}

func ShowMem() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %s; TotalAlloc = %s; Sys = %s;\n",
		ByteToKb(m.Alloc), ByteToKb(m.TotalAlloc), ByteToKb(m.Sys))
}
