package agent

import (
	"bufio"
	"os"
	"runtime"
	"strings"
	"sync"
)

var (
	runningInContainerOnce sync.Once
	runningInContainer     bool
)

// gets if the current process is running inside a container
func isRunningInContainer() bool {
	runningInContainerOnce.Do(func() {
		if runtime.GOOS == "linux" {
			file, err := os.Open("/proc/1/cgroup")
			if err != nil {
				runningInContainer = false
				return
			}
			defer file.Close()
			for {
				line, readErr := bufio.NewReader(file).ReadString('\n')
				if readErr != nil {
					break
				}
				if strings.Contains(line, "/docker/") || strings.Contains(line, "/lxc/") {
					runningInContainer = true
					return
				}
			}
		}
		runningInContainer = false
	})
	return runningInContainer
}
