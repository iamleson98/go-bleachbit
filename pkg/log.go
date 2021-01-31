package pkg

import (
	"os"
	"strings"
)

func isDebuggingEnabledViaCli() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--debug") {
			return true
		}
	}

	return false
}
