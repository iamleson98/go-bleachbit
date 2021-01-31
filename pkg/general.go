package pkg

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func makeDirs(path string) {
	log.Printf("Make dirs(%s)\n", path)
	if ext, _ := lExists(path); ext {
		return
	}

	parentDir, _ := filepath.Split(path)
	if !itemExist(parentDir) {
		makeDirs(parentDir)
	}

	err := os.Mkdir(path, 0o700) // 448
	if err != nil {
		log.Fatalf("Unable to make directory %s", path)
	}

	if sudoMode() {
		chownself(path)
	}
}

func sudoMode() bool {
	if LINUX != runtime.GOOS {
		return false
	}

	return os.Getenv("SUDO_UID") != ""
}

// getRealUID gets real user ID when running in sudo mode
func getRealUID() int {
	if LINUX != runtime.GOOS {
		log.Fatalln("getRealUID() requires POSIX")
	}

	if "" != os.Getenv("SUDO_UID") {
		intSudoUID, err := strconv.Atoi(os.Getenv("SUDO_UID"))
		if err != nil {
			log.Fatalln("Error converting SUDO_UID environment variable to integer")
		}
		return intSudoUID
	}

	login, err := getLogin()
	if err != nil {
		login = os.Getenv("LOGNAME")
	}

	if "" != login && "root" != login {
		pass, err := getPwNam(login)
		// NOTE: not sure
		if err == nil {
			return pass.pwUid
		}
		return os.Getuid()
	}

	return os.Getuid()
}

// chownself sets path owner to real self when running in sudo
func chownself(path string) {
	if LINUX != runtime.GOOS {
		return
	}

	uid := getRealUID()
	log.Printf("chown(%s, uid=%d)\n", path, uid)
	if 0 == strings.Index(path, "/root") {
		log.Println("chown for path /root aborted")
		return
	}

	err := os.Chown(path, uid, -1)
	if err != nil {
		log.Println("Error in chown() under chownself()")
	}
}
