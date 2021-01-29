package pkg

import (
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func isRunning(exename string) bool {
	if runtime.GOOS == "linux" {
		return isRunningLinux(exename)
	} else if runtime.GOOS == "darwin" ||
		strings.HasPrefix(runtime.GOOS, "openbsd") ||
		strings.HasPrefix(runtime.GOOS, "freebsd") {
		return isRunningDarwin(exename)
	}
	panic("unsupported platform for physical_free()")
}

func runProcess() string {
	cmd := exec.Command("ps", "aux", "-c")
	out, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}

	return string(out)
}

func isRunningDarwin(exename string) bool {
	// psResult := runProcess()
	// splitResult := strings.Split(psResult, "\n")
	// for

	panic("not implemented")
}

func isRunningLinux(exename string) bool {
	matches, err := filepath.Glob("/proc/*/exe")
	if err != nil {
		log.Println(err)
		return false
	}
	for _, filename := range matches {
		target, err := filepath.Abs(filename)
		if err != nil { // err can be os.ErrPermission
			continue
		}

		foundExename := filepath.Base(target)
		foundExename = strings.ReplaceAll(foundExename, " (deleted)", "")
		if exename == foundExename {
			return true
		}
	}

	return false
}
