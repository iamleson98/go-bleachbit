package pkg

import (
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

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
