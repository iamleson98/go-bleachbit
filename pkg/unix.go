package pkg

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	shlex "github.com/anmitsu/go-shlex"
	log "github.com/sirupsen/logrus"
	ini "gopkg.in/ini.v1"
)

var (
	// runtime check
	_ LocaleCleanerPathInterface = NewLocaleCleanerPath(&regexp.Regexp{})
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
		log.WithField("spot", "unix.isRunningLinux()").Errorln(err.Error())
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

type LocaleCleanerPath struct {
	pattern  *regexp.Regexp
	children []*regexp.Regexp
}

type LocaleCleanerPathInterface interface {
	addChild(child *regexp.Regexp) *regexp.Regexp
	addPathFilter(pre, post string)
	getSubpaths(basepath string) []string
	// getLocalizations(basepath string)
}

func NewLocaleCleanerPath(location *regexp.Regexp) *LocaleCleanerPath {
	if location == nil {
		log.WithField("spot", "unix.NewLocaleCleanerPath()").Fatalln("location is nil")
	}
	localeCleanerPath := new(LocaleCleanerPath)
	if location != nil {
		localeCleanerPath.pattern = location
	}

	return localeCleanerPath
}

func (lcp *LocaleCleanerPath) addChild(child *regexp.Regexp) *regexp.Regexp {
	lcp.children = append(lcp.children, child)
	return child
}

func (lcp *LocaleCleanerPath) addPathFilter(pre, post string) {
	exp, err := regexp.Compile("^" + pre + LocalePattern + post + "$")
	if err != nil {
		log.WithField("spot", "unix.LocaleCleanerPath.addPathFilter()").Fatalln(err.Error())
	}

	lcp.addChild(exp)
}

func (lcp *LocaleCleanerPath) getSubpaths(basepath string) []string {
	res := []string{}
	items, err := ioutil.ReadDir(basepath)
	if err != nil {
		log.WithField("spot", "unix.LocaleCleanerPath.getSubpaths()").Fatalln(err.Error())
	}

	for _, item := range items {
		if stat, err := os.Stat(filepath.Join(basepath, item.Name())); stat.IsDir() && err == nil && lcp.pattern.Match([]byte(item.Name())) {
			fullPath := filepath.Join(basepath, item.Name())
			res = append(res, fullPath)
		}
	}

	return res
}

// func (lcp *LocaleCleanerPath) getLocalizations(basepath string) {
// 	for _, path := range lcp.getSubpaths(basepath) {
// 		for _, child := range lcp.children {
// 			if items, err := ioutil.ReadDir(path); err == nil {
// 				for _, item := range items {

// 				}
// 			} else {
// 				log.WithField("spot", "unix.LocaleCleanerPath.getLocalizations()").Fatalln(err.Error())
// 			}
// 		}
// 	}
// }

func wineToLinuxPath(wineprefix, windowsPathname string) string {
	driveLetter := string(windowsPathname[0])
	windowsPathname = strings.ReplaceAll(
		windowsPathname,
		driveLetter+":",
		"drive_"+strings.ToLower(driveLetter),
	)
	windowsPathname = strings.ReplaceAll(windowsPathname, "\\", "/")

	return filepath.Join(wineprefix, windowsPathname)
}

func isBrokenXdgDesktopApplication(config *ini.File, desktopPathname string) bool {
	if key, err := config.Section("Desktop Entry").GetKey("Exec"); err != nil {
		log.WithField("spot", "unix.isBrokenXdgDesktopApplication()").Infoln("Missing required option 'Exec': " + desktopPathname)
		return true
	} else {
		val := key.Value()
		exe := strings.Split(val, " ")[0]
		if exeExists(exe) {
			log.WithField("spot", "unix.isBrokenXdgDesktopApplication()").Infoln("executable " + exe + " does not exist")
			return true
		}

		if "env" == exe {
			execs, err := shlex.Split(val, true)
			if err != nil {
				log.WithField("spot", "unix.isBrokenXdgDesktopApplication()").Errorln(err.Error())
			}
			winePrefix := ""
			for {
				if strings.Index(execs[0], "=") >= 0 {
					splitExecs := strings.Split(execs[0], "=")
					name := splitExecs[0]
					value := splitExecs[1]
					if "WINEPREFIX" == name {
						winePrefix = value
					}
				} else {
					break
				}
			}

			if !exeExists(execs[0]) {
				log.WithField("spot", "unix.isBrokenXdgDesktopApplication()").Infoln("executable does not exists")
				return true
			}

			if winePrefix != "" {
				windowsExe := wineToLinuxPath(winePrefix, execs[1])
				if !itemExist(windowsExe) {
					log.WithField("spot", "unix.isBrokenXdgDesktopApplication()").Infoln("Windows executable does not exist")
					return true
				}
			}
		}

		return false
	}
}
