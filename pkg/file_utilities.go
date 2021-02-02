package pkg

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yookoala/realpath"
)

var (
	whiteList whiteListed
)

// childrenInDirectory iterates through dir and make full path of all the items inside dir
// it should be called as a goroutine (mimic python's generator)
func childrenInDirectory(dir string, pathChan chan string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fullPath := filepath.Join(dir, path)
		pathChan <- fullPath

		return nil
	})
	close(pathChan)

	if err != nil {
		log.Fatal(err.Error())
	}
}

func openFileLinux() []string {
	matches, err := filepath.Glob("/proc/*/fd/*")
	if err != nil {
		log.Fatalln(err.Error())
	}
	return matches
}

func openFileLsof() []string {
	cmd := exec.Command("lsof", "-Fn", "-n")
	runResult, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}

	res := []string{}
	strRunResult := string(runResult)
	for _, f := range strings.Split(strRunResult, "\n") {
		if strings.HasPrefix(f, "n/") {
			res = append(res, f[1:])
		}
	}

	return res
}

func openFiles() []string {
	var files []string
	if LINUX == runtime.GOOS {
		files = openFileLinux()
	} else if DARWIN == runtime.GOOS {
		files = openFileLsof()
	} else {
		log.Fatalln("unsupported platform for openFiles()")
	}

	var res []string
	for _, filename := range files {
		realPath, err := realpath.Realpath(filename)
		if err != nil {
			continue
		}
		res = append(res, realPath)
	}

	return res
}

type whiteListed func(path string) bool

// func whiteListedWindows(path string) bool {

// }

// func whiteListedPosix(path string, checkRealPath bool) bool {

// }

func guessOverritePaths() []string {
	ret := []string{}

	if LINUX == runtime.GOOS {
		home := ExpandUser("~/.cache")
		if !itemExist(home) {
			home = ExpandUser("~")
		}
		ret = append(ret, home)

	} else if WINDOWS == runtime.GOOS {
		// localTmp := os.ExpandEnv("$TMP")
		// if !itemExist(localTmp) {
		// 	log.Println("%TMP% does not exist")
		// }
		panic("not implemented for windows")
	} else {
		log.Fatalln("Unsupported os in guessOverritePaths")
	}

	return ret
}

// func samePartition(dir1, dir2 string) bool {
// 	if WINDOWS == runtime.GOOS {
// 		panic("not implemented")
// 	}

// 	// disk, _ := diskfs.Open("dfdf")
// 	// disk.GetFilesystem()
// 	// disk.GetPartitionTable()

// }

// func freeSpace(pathname string) {
// 	if WINDOWS == runtime.GOOS {
// 		// NOTE: This part does not follow the original implementation in python
// 		// majorVersion := parseWindowsBuild(nil)[0]
// 		// if majorVersion >= 6 {

// 		// }
// 		h := syscall.MustLoadDLL("kernel32.dll")
// 		c := h.MustFindProc("GetDiskFreeSpaceExW")

// 		var freeBytes int64
// 		_, _, err := c.Call(
// 			uintptr(
// 				unsafe.Pointer(syscall.StringToUTF16Ptr(pathname)),
// 			),
// 			uintptr(
// 				unsafe.Pointer(&freeBytes),
// 			),
// 			nil,
// 			nil,
// 		)

// 		return freeBytes
// 	}

// 	var stat syscall.Statfs_t
// 	err := syscall.Statfs(pathname, &stat)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return stat.Bavail * uint64(stat.Bsize)
// }

type openFilesStruct struct {
	lastScanTime *time.Time
	files        []string
}

func newOpenFilesStruct() *openFilesStruct {
	return new(openFilesStruct)
}

func (o *openFilesStruct) fileQualifies(filename string) bool {
	return !strings.HasPrefix(filename, "/dev") && !strings.HasPrefix(filename, "/proc")
}

func (o *openFilesStruct) scan() {
	now := time.Now()
	o.lastScanTime = &now
	o.files = []string{}

	for _, fName := range openFiles() {
		if o.fileQualifies(fName) {
			o.files = append(o.files, fName)
		}
	}
}

func (o *openFilesStruct) isOpen(filename string) bool {
	if o.lastScanTime == nil || o.lastScanTime.Add(10*time.Second).Before(time.Now()) {
		o.scan()
	}
	realPath, err := realpath.Realpath(filename)
	if err != nil {
		log.Fatalln(err.Error())
	}

	for _, fName := range o.files {
		if fName == realPath {
			return true
		}
	}

	return false
}

// func bytesToHuman(bytes int) string {
// 	if bytes < 0 {
// 		return "-" + bytesToHuman(-bytes)
// 	}

// }

func existsInPath(filename string) bool {
	delimiter := ":"
	if WINDOWS == runtime.GOOS {
		delimiter = ";"
	}

	for _, dirname := range strings.Split(os.Getenv("PATH"), delimiter) {
		if itemExist(filepath.Join(dirname, filename)) {
			return true
		}
	}

	return false
}

func exeExists(pathname string) bool {
	if filepath.IsAbs(pathname) {
		return itemExist(pathname)
	}
	return existsInPath(pathname)
}

func egoOwner(filename string) bool {
	if info, err := os.Lstat(filename); err != nil {
		return false
	} else {
		sys := info.Sys()
		if stat, ok := sys.(*syscall.Stat_t); ok {
			UID := int(stat.Uid)
			return UID == os.Getuid()
		} else {
			log.WithField("spot", "file_utilities.egoOwner()").Errorln("Cannot get owner ID of file")
			return true
		}
	}
}

func globex(pathname string, regex *regexp.Regexp) {

}
