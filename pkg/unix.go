package pkg

import (
	"fmt"
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

// dds a filter consisting of a prefix and a postfix
func (lcp *LocaleCleanerPath) addPathFilter(pre, post string) {
	exp, err := regexp.Compile("^" + pre + LocalePattern + post + "$")
	if err != nil {
		log.WithField("spot", "unix.LocaleCleanerPath.addPathFilter()").Fatalln(err.Error())
	}

	lcp.addChild(exp)
}

// Returns direct subpaths for this object
func (lcp *LocaleCleanerPath) getSubpaths(basepath string) []string {
	res := []string{}
	items, err := ioutil.ReadDir(basepath)
	if err != nil {
		log.WithField("spot", "unix.LocaleCleanerPath.getSubpaths()").Fatalln(err.Error())
	}

	for _, item := range items {
		fullPath := filepath.Join(basepath, item.Name())
		if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() && lcp.pattern.MatchString(item.Name()) {
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

// Returns boolean whether application desktop entry file is broken
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
			// Wine v1.0 creates .desktop files like this
			// Exec=env WINEPREFIX="/home/z/.wine" wine "C:\\Program
			// Files\\foo\\foo.exe"
			execs, err := shlex.Split(val, true)
			if err != nil {
				log.WithField("spot", "unix.isBrokenXdgDesktopApplication()").Errorln(err.Error())
			}
			var winePrefix string
			execs = execs[1:]
			for {
				if strings.Index(execs[0], "=") >= 0 {
					splitExecs := strings.Split(execs[0], "=")
					name := splitExecs[0]
					value := splitExecs[1]
					if "WINEPREFIX" == name {
						winePrefix = value
					}
					execs = execs[1:]
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

// Yield a list of rotated (i.e., old) logs in /var/log/
// func rotatedLogs() []string {
// 	var res []string
// 	globPaths := []string{
// 		"/var/log/*.[0-9]",
// 		"/var/log/*/*.[0-9]",
// 		"/var/log/*.gz",
// 		"/var/log/*/*gz",
// 		"/var/log/*/*.old",
// 		"/var/log/*.old",
// 	}
// 	for _, globPath := range globPaths {
// 		matches, err := filepath.Glob(globPath)
// 		if err != nil {
// 			log.WithField("spot", "unix.rotatedLogs()").Fatalln(err.Error())
// 			return nil
// 		}
// 		res = append(res, matches...)
// 	}
// 	regex := regexp.MustCompile("-[0-9]{8}$")
// 	globPaths = []string{"/var/log/*-*", "/var/log/*/*-*"}
// 	whitelistRe := regexp.MustCompile("^/var/log/(removed_)?(packages|scripts)")
// 	for _, glob := range globPaths {
// 		globex_ := globex(glob, regex)
// 		if globex_ != nil {
// 			for _, gl := range globex_ {

// 			}
// 		}
// 	}
// }

func runCleanerCmd(cmd string, args []string, freedSpace string, errLines []string) int {

	if freedSpace == "" {
		freedSpace = `[\d.]+[kMGTE]?B?`
	}
	if !exeExists(cmd) {
		log.WithField("spot", "unix.runCleanerCmd").Fatalln("Executable not found: " + cmd)
	}

	freedSpaceReges := regexp.MustCompile(freedSpace)
	errorLineRegexes := []*regexp.Regexp{}
	for _, errLine := range errLines {
		errorLineRegexes = append(errorLineRegexes, regexp.MustCompile(errLine))
	}

	command := exec.Command(cmd, args...)
	// NOTE: not sure. refer to https://docs.python.org/3/library/subprocess.html#subprocess.run
	// whether to assign new or append to existing environment
	command.Env = []string{"LC_ALL=C", fmt.Sprintf("PATH=%s", os.Getenv("PATH"))}

	output, err := command.Output()
	if err != nil {
		log.WithField("spot", "unix.runCleanerCmd()").Fatalln(err.Error())
	}
	strOutput := string(output)

	freeSpace := 0
	for _, line := range strings.Split(strOutput, "\n") {
		matches := freedSpaceReges.FindStringSubmatch(line)
		if matches != nil {
			// NOTE: note sure yet
			freeSpace += humanToBytes(matches[0], "")
		}
		for _, errRe := range errorLineRegexes {
			if errRe.MatchString(line) {
				log.WithField("spot", "unix.runCleanerCmd()").Fatalf("Invalid output from %s: %s\n", cmd, line)
			}
		}
	}

	return freeSpace
}

func getAptSize() {

}

// func getGlobsSize(paths ...string) {
// 	totalSize := 0
// 	for _, path := range paths {
// 		matches, err := filepath.Glob(path)
// 		if err != nil {
// 			log.WithField("spot", "unix.getBlobsSize()").Fatalln(err.Error())
// 		}
// 		for _, match := range matches {
// 			totalSize +=
// 		}
// 	}
// }

func aptAutoRemove() int {
	args := []string{"--yes", "autoremove"}
	freedSpaceRegex := `.*, ([\d.]+ ?[a-zA-Z]{2}) disk space will be freed.`
	res := runCleanerCmd("apt-get", args, freedSpaceRegex, []string{"^E: "})
	return res
}

func aptAutoClean() int {
	res := runCleanerCmd("apt-get", []string{"autoclean"}, `^Del .*\[([\d.]+[a-zA-Z]{2})}]`, []string{"^E: "})
	return res
}

// func aptClean() {
// 	oldSize :=
// }
