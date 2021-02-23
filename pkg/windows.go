// +build windows

package pkg

import (
	"log"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func parseWindowsBuild(build *string) []int {
	if build == nil {
		return getWindowsVersion()
	}

	splitBuild := strings.Split(*build, ".")
	maj, min := splitBuild[0], splitBuild[1]

	intMaj, err := strconv.Atoi(maj)
	if err != nil {
		log.Fatal(err)
	}

	intMin, err := strconv.Atoi(min)
	if err != nil {
		log.Fatal(err)
	}

	return []int{intMaj, intMin}
}

// getWindowsVersion returns windows major and minor version like 10.0
func getWindowsVersion() []int {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		log.Fatal(err)
	}

	maj, _, err := k.GetIntegerValue("CurrentMajorVersionNumber")
	if err != nil {
		log.Fatal(err)
	}

	min, _, err := k.GetIntegerValue("CurrentMinorVersionNumber")
	if err != nil {
		log.Fatal(err)
	}

	return []int{maj, min}
}
