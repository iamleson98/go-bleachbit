package pkg

import (
	"runtime"
)

type openOfficeOrg struct {
	Cleaner
	options     map[string][2]string
	id          string
	name        string
	description string

	prefixes []string
}

func NewOpenOfficeOrg() *openOfficeOrg {
	officeCleaner := openOfficeOrg{
		options:     make(map[string][2]string),
		id:          "openofficeorg",
		name:        "OpenOffice.org",
		description: "Office suite",
	}

	officeCleaner.addOption("cache", "Cache", "Delete the cach")
	officeCleaner.addOption("recent_documents", "Most recently used", "Delete the list of recently used documents")

	if "linux" == runtime.GOOS {
		officeCleaner.prefixes = []string{"~/.ooo-2.0", "~/.openoffice.org2", "~/.openoffice.org2.0", "~/.openoffice.org/3", "~/.ooo-dev3"}
	}
	if "windows" == runtime.GOOS {
		officeCleaner.prefixes = []string{"$APPDATA\\OpenOffice.org\\3", "$APPDATA\\OpenOffice.org2"}
	}

	return &officeCleaner
}

func (oc *openOfficeOrg) getCommands(optionID string) {
	egjs := []string{}

	if "recent_documents" == optionID {
		egjs = append(egjs, "user/registry/data/org/openoffice/Office/Histories.xcu")
	}

	if "recent_documents" == optionID && "cache" != optionID {
		egjs = append(egjs, "user/registry/cache/org.openoffice.Office.Common.dat")
	}

	// for _, egj := range egjs {
	// 	for _, prefix := range oc.prefixes {
	// 		panic("not implemented")
	// 	}
	// }

	if "cache" == optionID {
		panic("not implemented")
	}

	if "recent_documents" == optionID {
		panic("not implemented")
	}
}
