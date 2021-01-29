package pkg

import (
	"log"
	"regexp"
	"runtime"
)

type system struct {
	Cleaner
	description string
	id          string
	name        string
}

func newSystemCleaner() *system {
	systemCleaner := new(system)

	if LINUX == runtime.GOOS {
		systemCleaner.addOption("desktop_entry", "Broken desktop files", "Delete broken application menu entries and file associations")
		systemCleaner.addOption("cache", "Cache", "Delete the cache")
		systemCleaner.addOption("localizations", "Localizations", "Delete files for unwanted languages")

		systemCleaner.setWarning("localizations", "Configure this options in the preferences.")

		systemCleaner.addOption("rotated_logs", "Rotated logs", "Delete old system logs")
		systemCleaner.addOption("recent_documents", "Recent documents list", "Delete the list of recently used documents")
		systemCleaner.addOption("trash", "Trash", "Empty the trash")

		systemCleaner.addOption("memory", "Memory", "Wipe the swap and free memory")
		systemCleaner.setWarning("memory", "This options is experimental and may cause system problems.")
	}

	if WINDOWS == runtime.GOOS {
		systemCleaner.addOption("logs", "Logs", "Delete the logs")
		systemCleaner.addOption("memory_dump", "Memory dump", "Delete the file")
		systemCleaner.addOption("muicache", "MUICache", "Delete the cache")
		systemCleaner.addOption("prefetch", "Prefetch", "Delete the cache")
		systemCleaner.addOption("recycle_bin", "Recycle bin", "Empty the recycle bin")
		systemCleaner.addOption("updates", "Update uninstallers", "Delete uninstallers for Microsoft updates including hotfixes, service packs, and Internet Explorer updates")
	}

	// not implement for systems that have GTK+
	// https://github.com/bleachbit/bleachbit/blob/e5076cf63ef0535fa4d629565580d6a08710c15f/bleachbit/Cleaner.py#L330

	systemCleaner.addOption("custom", "Custom", "Delete user-specified files and folders")
	systemCleaner.addOption("free_disk_space", "Free disk space", "Overwrite free disk space to hide deleted files")
	systemCleaner.setWarning("free_disk_space", "This option is very slow")
	systemCleaner.addOption("tmp", "Temporary files", "Delete the temporary files")

	systemCleaner.description = "The system in general"
	systemCleaner.id = "system"
	systemCleaner.name = "System"

	return systemCleaner
}

func (sc *system) getCommands(optionID string) {
	// panic("not implemented")
	if LINUX == runtime.GOOS && "cache" == optionID {
		dirName := ExpandUser("~/.cache/")

		c := make(chan string)
		go childrenInDirectory(dirName, c)

		for {
			_, ok := <-c
			if !ok {
				break
			}

		}
	}
}

func (sc *system) whiteListed(pathname string) bool {
	if WINDOWS == runtime.GOOS {
		return false
	}

	if len(sc.regexesCompiled) == 0 {
		sc.initWhiteList()
	}

	for _, reg := range sc.regexesCompiled {
		if reg.Match([]byte(pathname)) {
			return true
		}
	}

	return false
}

func (sc *system) initWhiteList() {

	regexes := []string{
		"^/tmp/.X0-lock$",
		"^/tmp/.truecrypt_aux_mnt.*/(control|volume)$",
		"^/tmp/.vbox-[^/]+-ipc/lock$",
		"^/tmp/.wine-[0-9]+/server-.*/lock$",
		"^/tmp/gconfd-[^/]+/lock/ior$",
		"^/tmp/fsa/", // fsarchiver
		"^/tmp/kde-",
		"^/tmp/kdesudo-",
		"^/tmp/ksocket-",
		"^/tmp/orbit-[^/]+/bonobo-activation-register[a-z0-9-]*.lock$",
		"^/tmp/orbit-[^/]+/bonobo-activation-server-[a-z0-9-]*ior$",
		"^/tmp/pulse-[^/]+/pid$",
		"^/var/tmp/kdecache-",
		"^" + ExpandUser("~/.cache/wallpaper/"),
		// Flatpak mount point
		"^" + ExpandUser("~/.cache/doc($|/)"),
		//  Clean Firefox cache from Firefox cleaner (LP#1295826)
		"^" + ExpandUser("~/.cache/mozilla/"),
		// Clean Google Chrome cache from Google Chrome cleaner (LP#656104)
		"^" + ExpandUser("~/.cache/google-chrome/"),
		"^" + ExpandUser("~/.cache/gnome-control-center/"),
		// Clean Evolution cache from Evolution cleaner (GitHub #249)
		"^" + ExpandUser("~/.cache/evolution/"),
		// iBus Pinyin
		// https://bugs.launchpad.net/bleachbit/+bug/1538919
		"^" + ExpandUser("~/.cache/ibus/"),
		// Linux Bluetooth daemon obexd directory is typically empty, so be careful
		// not to delete the empty directory.
		"^" + ExpandUser("~/.cache/obexd($|/)"),
	}

	for _, str := range regexes {
		regEx, err := regexp.Compile(str)
		if err != nil {
			log.Printf("Error occured: %v\n", err.Error())
			continue
		}

		sc.regexesCompiled = append(sc.regexesCompiled, regEx)
	}
}
