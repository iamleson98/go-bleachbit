package pkg

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	_UNIXCONFDIR = "/etc"
)

func distTryHarder(distName, version, id string) (string, string, string) {
	if itemExist("/var/adm/inst-log/info") {
		distName = "SuSE"
		data, err := ioutil.ReadFile("/var/adm/inst-log/info")
		if err != nil {
			log.WithField("spot", "platform.distTryHarder()").Fatalln(err.Error())
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			var tag, value string
			line := scanner.Text()
			splitLine := strings.Split(line, " ")
			if len(splitLine) == 2 {
				tag, value = splitLine[0], splitLine[1]
			} else {
				continue
			}

			if tag == "MIN_DIST_VERSION" {
				version = strings.TrimSpace(value)
			} else if tag == "DIST_IDENT" {
				values := strings.Split(value, "-")
				id = values[2]
			}
		}

		return distName, version, id
	}

	if itemExist("/etc/.installed") {
		data, err := ioutil.ReadFile("/etc/.installed")
		if err != nil {
			log.WithField("spot", "platform.distTryHarder()").Fatalln(err.Error())
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			pkg := strings.Split(scanner.Text(), "-")
			if len(pkg) >= 2 && pkg[0] == "OpenLinux" {
				return "OpenLinux", pkg[1], id
			}
		}
	}

	stat, err := os.Stat("/usr/lib/setup")
	if os.IsNotExist(err) {
		return distName, version, id
	}

	if stat.IsDir() {
		items, err := ioutil.ReadDir("/usr/lib/setup")
		if err != nil {
			return distName, version, id
		}

		for n := len(items); n >= 0; n-- {
			if items[n].Name()[:14] == "slack-version-" {
				items = append(items[:n], items[n+1:]...)
			}
		}

		if len(items) > 0 {
			sort.Slice(items, func(i, j int) bool {
				return items[i].Name() < items[j].Name()
			})

			distName = "slackware"
			version = items[len(items)-1].Name()[14:]
			return distName, version, id
		}
	}

	return distName, version, id
}
