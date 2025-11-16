package debug

import (
	"fmt"
	"runtime"
	"time"
)

func EnableDumpGoroutines(duration time.Duration) {
	go func() {
		for {
			time.Sleep(duration)
			dumpGoroutines()
		}
	}()
}

func dumpGoroutines() {
	buf := make([]byte, 1<<20) // 1 MB
	n := runtime.Stack(buf, true)
	fmt.Printf("=== Goroutines ===\n%s\n=== End ===\n", buf[:n])
}
