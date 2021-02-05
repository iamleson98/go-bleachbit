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
	"github.com/h2non/filetype"
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

func isRunningDarwin(exename string) bool {
	out, err := exec.Command("ps", "aux", "-c").Output()
	if err != nil {
		log.WithField("spot", "unix.isRunningDarwin()").Fatalln(err.Error())
	}

	strOut := string(out)
	splitStrOut := strings.Split(strOut, "\n")
	regExp := regexp.MustCompile(`\s+`)

	processes := []string{}
	for _, p := range splitStrOut {
		if p != "" {
			list := regExp.Split(p, 10)
			if len(list) >= 11 {
				processes = append(processes, list[10])
			} else {
				log.WithField("spot", "unix.isRunningDarwin()").Errorln("Unexpected output from ps")
			}
		}
	}

	// first line is result table header, no need
	processes = processes[1:]

	return valueInList(exename, &processes)
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

// check if a mimetype is not supported
func isUnregisteredMime(mime string) bool {
	if filetype.IsMIMESupported(mime) {
		return false
	}
	return true
}

// Returns boolean whether the given XDG desktop entry file is broken.
// Reference: http://standards.freedesktop.org/desktop-entry-spec/latest/
func isBrokenXdgDesktop(pathname string) bool {
	iniConfig, err := ini.Load(pathname)
	if err != nil {
		log.WithField("spot", "unix.isBrokenXdgDesktop()").Fatalln("Configuration file " + pathname + " does not exist")
		return true
	}

	if sec, err := iniConfig.GetSection("Desktop Entry"); err != nil {
		log.WithField("spot", "unix.isBrokenXdgDesktop()").Infoln("missing required section 'Desktop Entry' " + pathname)
		return true
	} else {
		if !sec.HasKey("Type") {
			log.WithField("spot", "unix.isBrokenXdgDesktop()").Infoln("missing required option 'Entry' " + pathname)
			return true
		}

		fileType := strings.TrimSpace(sec.Key("Type").Value())
		fileType = strings.ToLower(fileType)

		if "link" == fileType {
			if !sec.HasKey("URL") && !sec.HasKey("URL[$e]") {
				log.WithField("spot", "unix.isBrokenXdgDesktop()").Infoln("missing required option 'URL': " + pathname)
				return true
			}
			return false
		}

		if "mimetype" == fileType {
			if !sec.HasKey("MimeType") {
				log.WithField("spot", "unix.isBrokenXdgDesktop()").Infoln("missing required option 'MimeType': " + pathname)
				return true
			}

			mimeType := strings.TrimSpace(sec.Key("MimeType").Value())
			mimeType = strings.ToLower(mimeType)
			if isUnregisteredMime(mimeType) {
				log.WithField("spot", "unix.isBrokenXdgDesktop()").Infof("MimeType %s is not supported %s\n", mimeType, pathname)
				return true
			}
			return false
		}

		if "application" != fileType {
			log.WithField("spot", "unix.isBrokenXdgDesktop()").Warningf("unhandled type '%s': file '%s'\n", fileType, pathname)
			return false
		}

		if isBrokenXdgDesktopApplication(iniConfig, pathname) {
			return true
		}

		return false
	}
}

// func runCleanerCmd(cmd string, args []string, freedSpaceRegex string, errorLineRegexes []string) {

// 	if freedSpaceRegex == "" {
// 		freedSpaceRegex = `[\d.]+[kMGTE]?B?`
// 	}
// 	if !exeExists(cmd) {
// 		log.WithField("spot", "unix.runCleanerCmd").Fatalln("Executable not found: " + cmd)
// 	}

// 	freedSpaceRegex_ := regexp.MustCompile(freedSpaceRegex)
// 	errorLineRegexes_ := []*regexp.Regexp{}
// 	for _, strErrRegex := range errorLineRegexes {
// 		errorLineRegexes_ = append(errorLineRegexes_, regexp.MustCompile(strErrRegex))
// 	}

// 	command := exec.Command(cmd, args...)
// 	// NOTE: not sure. refer to https://docs.python.org/3/library/subprocess.html#subprocess.run
// 	// whether to assign new or append to existing environment
// 	command.Env = []string{"LC_ALL=C", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}

// 	output, err := command.Output()
// 	if err != nil {
// 		log.WithField("spot", "unix.runCleanerCmd()").Fatalln(err.Error())
// 	}
// 	strOutput := string(output)
// 	splitOutput := strings.Split(strOutput, "\n")

// 	freedSpace := 0
// 	for _, line := range splitOutput {
// 		freedSpaceRegex_
// 	}
// }

func aptAutoClean() {

}
