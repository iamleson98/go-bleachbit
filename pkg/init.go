package pkg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp/syntax"
	"runtime"
)

var (
	optionsDir        string
	bleachbitExePath  string
	socketTimeout     int  = 10
	portableMode      bool = false
	optionsFile       string
	systemCleanersDir string
	localCleanersDir  string
	fsScanReFlags     syntax.Flags
	userLocale        string
	encoding          string
	appMenuFileName   string
	localeDir         string
)

const (
	APP_VERSION = "1.0.0"
	APP_NAME    = "BleachBit"
	APP_URL     = "https://www.bleachbit.org"
	// LINUX if os is linux
	LINUX string = "linux"

	// WINDOWS if os is windows
	WINDOWS string = "windows"

	// DARWIN
	DARWIN string = "darwin"
)

func init() {

	// get dirname of executable file
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Error running program %v", err)
	}
	bleachbitExePath = filepath.Dir(exePath)

	if LINUX == runtime.GOOS {
		optionsDir = ExpandUser("~/.config/bleachbit")
	} else if WINDOWS == runtime.GOOS {
		err := os.Unsetenv("FONTCONFIG_FILE")
		if err != nil {
			log.Printf("Error unset env: %v", err.Error())
		}

		exist := itemExist(filepath.Join(bleachbitExePath, "bleachbit.ini"))
		if exist { // means file exists
			portableMode = true
			optionsDir = bleachbitExePath
		} else {
			optionsDir = os.ExpandEnv("${APPDATA}\\BleachBit")
		}
	}

	optionsFile = filepath.Join(optionsDir, "bleachbit.ini")

	if !portableMode {
		e1 := itemExist(filepath.Join(bleachbitExePath, "../cleaners"))
		e2 := itemExist(filepath.Join(bleachbitExePath, "../Makefile"))
		e3 := itemExist(filepath.Join(bleachbitExePath, "../COPYING"))
		portableMode = e1 && e2 && e3
	}

	info, err := os.Stat(filepath.Join(bleachbitExePath, "cleaners"))
	if err == nil && info.IsDir() && !portableMode {
		systemCleanersDir = filepath.Join(bleachbitExePath, "cleaners")
	}
	if LINUX == runtime.GOOS || DARWIN == runtime.GOOS {
		systemCleanersDir = "/usr/share/bleachbit/cleaners"
	} else if WINDOWS == runtime.GOOS {
		systemCleanersDir = filepath.Join(bleachbitExePath, "/share/cleaners/")
	} else {
		systemCleanersDir = ""
		log.Printf("Unknown system cleaners directory for platform %s\n", runtime.GOOS)
	}

	if portableMode {
		localCleanersDir = filepath.Join(bleachbitExePath, "cleaners")
	}

	if LINUX == runtime.GOOS {
		fsScanReFlags = syntax.POSIX
	} else {
		fsScanReFlags = syntax.FoldCase
	}
}

func init() {
	userLocale, encoding = getDefaultLocale()

	if userLocale == "" {
		userLocale = "C"
		log.Println("no default locale found. Assume " + userLocale)
	}

	if WINDOWS == runtime.GOOS {
		os.Setenv("LANG", userLocale)
	}
}

func init() {
	appMenuFileName = filepath.Join(bleachbitExePath, "data", "app-menu.ui")
	if !itemExist(appMenuFileName) {
		appMenuFileName, _ = filepath.Abs(
			filepath.Join(systemCleanersDir, "../app-menu.ui"),
		)
	}

	if !itemExist(appMenuFileName) {
		log.Println("unknown location for app-menu.ui")
	}
}

func init() {
	if itemExist("./locale/") {
		localeDir, _ = filepath.Abs("./locale/")
	} else {
		if LINUX == runtime.GOOS || DARWIN == runtime.GOOS {
			localeDir = "/usr/share/locale/"
		} else if WINDOWS == runtime.GOOS {
			localeDir = filepath.Join(bleachbitExePath, "share\\locale\\")
		}
	}

	if !itemExist(localeDir) {
		log.Println("translations not installed")
	}
}

var baseUrl = "https://update.bleachbit.org"
var helpContentsUrl = fmt.Sprintf("%s/help/%s", baseUrl, APP_VERSION)
var releaseNodeUrl = fmt.Sprintf("%s/release-note/%s", baseUrl, APP_VERSION)
var updateCheckUrl = fmt.Sprintf("%s/update/%s", baseUrl, APP_VERSION)

func init() {
	if WINDOWS == runtime.GOOS {

	}

	if LINUX == runtime.GOOS {
		envs := map[string]string{
			"XDG_DATA_HOME":   ExpandUser("~/.local/share"),
			"XDG_CONFIG_HOME": ExpandUser("~/.config"),
			"XDG_CACHE_HOME":  ExpandUser("~/.cache"),
		}

		for key, val := range envs {
			if os.Getenv(val) == "" {
				os.Setenv(key, val)
			}
		}
	}
}
