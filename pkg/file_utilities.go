package pkg

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yookoala/realpath"
)

var (
	whiteList whiteListed

	// runtime type checking
	_ openFilesInterface = newOpenFilesStruct()
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

type openFilesInterface interface {
	fileQualifies(filename string) bool
	scan()
	isOpen(filename string) bool
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

// Display a file size in human terms (megabytes, etc.) using preferred standard (SI or IEC)
func bytesToHuman(bytes int) string {
	if bytes < 0 {
		return "-" + bytesToHuman(-bytes)
	}

	if bytes == 0 {
		return "0"
	}

	var prefixes []string
	var base float64
	var decimals string

	if options_.get("units_iec", "", getBool) == true {
		prefixes = []string{"", "Ki", "Mi", "Gi", "Ti", "Pi"}
		base = 1024
	} else {
		prefixes = []string{"", "k", "M", "G", "T", "P"}
		base = 1000
	}

	if float64(bytes) >= math.Pow(base, 3) {
		decimals = "%.2f"
	} else if float64(bytes) > base {
		decimals = "%.1f"
	} else {
		decimals = "%.f"
	}

	for _, p := range prefixes {
		if float64(bytes) < base {
			abbrev := fmt.Sprintf(decimals, float64(bytes))
			suf := p
			return abbrev + suf + "B"
		} else {
			bytes = int(float64(bytes) / base)
		}
	}

	return "A lot"
}

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

func expandGlobJoin(pathname1, pathname2 string) []string {
	pathname3 := ExpandUser(os.ExpandEnv(filepath.Join(pathname1, pathname2)))
	if matches, err := filepath.Glob(pathname3); err != nil {
		log.WithField("spot", "file_utilities.expandGlobJoin()").Fatalln("Error finding glob")
		return []string{}
	} else {
		return matches
	}
}

// If applicable, return the extended Windows pathname
func extendedPath(path string) string {
	if WINDOWS == runtime.GOOS {
		if strings.HasPrefix(path, "\\?") {
			return path
		}
		if strings.HasPrefix(path, "\\") {
			return "\\\\?\\unc\\" + path[2:]
		}
		return "\\\\?\\" + path
	}

	return path
}

func extendedPathUndo(path string) string {
	if WINDOWS == runtime.GOOS {
		if strings.HasPrefix(path, `\\?\unc`) {
			return "\\" + path[7:]
		}
		if strings.HasPrefix(path, `\\?`) {
			return path[4:]
		}
	}

	return path
}

func globex(pathname string, regex *regexp.Regexp) {

}

func humanToBytes(human string, hformat string) int {

	var base float64
	var suffixes string

	if hformat == "" {
		hformat = "si"
	}

	if hformat == "si" {
		base = 1000
		suffixes = "kMGTE"
	} else if hformat == "du" {
		base = 1024
		suffixes = "KMGTE"
	} else {
		log.WithField("spot", "file_utilities.humanToBytes()").Fatalln("Invalid format: " + hformat)
	}

	reg := regexp.MustCompile(fmt.Sprintf(`^(\d+(?:\.\d+)?) ?([%s]?)B?$`, suffixes))
	matches := reg.FindStringSubmatch(human)
	// matches should has length of 3 if matches
	// E.g human = 10GB => matches == []string{"10GB", "10", "G"}
	if matches == nil {
		log.WithField("spot", "file_utilities.humanToBytes()").Fatalf("Invalid input for '%s' (hformat='%s')", human, hformat)
	}

	amount, suffix := matches[1], matches[2]
	var exponent int
	if "" == suffix {
		exponent = 0
	} else {
		exponent = strings.Index(suffixes, suffix) + 1
	}

	float64Amount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		log.WithField("spot", "file_utilities.humanToBytes()").Fatalln(err.Error())
	}

	return int(math.Pow(base, float64(exponent)) * float64Amount)
}
