package pkg

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	ini "gopkg.in/ini.v1"
)

var (
	booleanKeys []string
	intKeys     []string
	options_    *options
)

func init() {
	booleanKeys = []string{
		"auto_hide",
		"check_beta",
		"check_online_updates",
		"dark_mode",
		"delete_confirmation",
		"debug",
		"exit_done",
		"first_start",
		"shred",
		"units_iec",
		"window_maximized",
		"window_fullscreen",
	}

	intKeys = []string{
		"window_x",
		"window_y",
		"window_width",
		"window_height",
	}

	options_ = newOptions()
	if WINDOWS == runtime.GOOS {
		booleanKeys = append(booleanKeys, "update_winapp32", "win10_theme")
	}
}

func pathToOption(pathname string) string {
	if WINDOWS == runtime.GOOS && itemExist(pathname) {
		panic("not implemented")
	}

	if char := pathname[1]; string(char) == ":" {
		pathname = string(pathname[0]) + pathname[2:]
	}

	return pathname
}

func initConfiguration() {
	if !itemExist(optionsDir) {
		makeDirs(optionsDir)
	}

	if _, err := os.Lstat(optionsFile); err != nil {
		log.Printf("Deleting configuration: %s", optionsFile)
		err := os.Remove(optionsFile)
		log.Println(err.Error())
	}

	fIni, err := os.Create(optionsFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer fIni.Close()

	_, err = fIni.WriteString("[bleachbit]\n")
	if err != nil {
		log.Fatalln(err.Error())
	}
	if WINDOWS == runtime.GOOS && portableMode {
		fIni.WriteString("[Portable]\n")
	}

	for _, section := range options_.config.Sections() {
		options_.config.DeleteSection(section.Name())
	}

	options_.restore()
}

type options struct {
	purged bool
	config *ini.File
}

func newOptions() *options {
	opts := new(options)
	opts.purged = false
	opts.config = new(ini.File)

	defer opts.restore()

	return opts
}

func (o *options) flush() {
	if !o.purged {
		o.purge()
	}

	if !itemExist(optionsDir) {
		makeDirs(optionsDir)
	}

	mkFile := !itemExist(optionsFile)

	// save all configs to optionsFile
	err := o.config.SaveTo(optionsFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	if mkFile && sudoMode() {
		chownself(optionsFile)
	}
}

func (o *options) purge() {
	o.purged = true
	if hashPathSec, err := o.config.GetSection("hashpath"); err != nil {
		// return if section named "hashpath" does not exist
		return
	} else {
		alphaNumericRe := regexp.MustCompile("^[a-z]\\\\")
		for _, sec := range hashPathSec.ChildSections() {
			optName := sec.Name()
			pathName := optName
			if WINDOWS == runtime.GOOS && alphaNumericRe.Match([]byte(optName)) {
				// restore colon lost because ConfigParser treats colon special in keys
				pathName = string(pathName[0]) + ":" + string(pathName[1:])
			}

			exists := false
			if ext, err := lExists(pathName); err == nil {
				exists = ext
			} else {
				log.Println("Error checking whether [ath exists")
			}

			if !exists {
				hashPathSec.DeleteKey(optName)
			}
		}
	}
}

func (o *options) setDefault(key, value string) {
	_, err := o.config.Section("bleachbit").NewKey(key, value)
	if err != nil {
		log.Fatalln(err.Error())
	}
}

func (o *options) set(key, value, section string, commit bool) {
	if section == "" {
		section = "bleachbit"
	}
	sec := o.config.Section(section)
	if k, err := sec.GetKey(key); err != nil {
		_, err := sec.NewKey(key, value)
		if err != nil {
			log.Fatalln(err.Error())
		}
	} else {
		k.SetValue(value)
	}

	if commit {
		o.flush()
	}
}

func (o *options) commit() {
	o.flush()
}

// restore performs restoring saved options from disk
func (o *options) restore() {
	cfg, err := ini.Load(optionsFile)
	if err != nil {
		log.Printf("Error reading application's configuration %s\n", err.Error())
	}

	o.config = cfg

	if _, err := o.config.GetSection("bleachbit"); err != nil {
		o.config.NewSection("bleachbit") // NOTE: noqa
	}
	if _, err := o.config.GetSection("hashpath"); err != nil {
		o.config.NewSection("hashpath") // NOTE: noqa
	}
	if _, err := o.config.GetSection("list/shred_drives"); err != nil {
		guessOvrPaths := guessOverritePaths()
		err := o.setList("shred_drives", guessOvrPaths)
		if err != nil {
			log.Println(err.Error())
		}
	}

	// set defaults
	o.setDefault("auto_hide", "true")
	o.setDefault("check_beta", "false")
	o.setDefault("check_online_updates", "true")
	o.setDefault("dark_mode", "true")
	o.setDefault("delete_confirmation", "true")
	o.setDefault("debug", "false")
	o.setDefault("exit_done", "false")
	o.setDefault("shred", "false")
	o.setDefault("units_iec", "false")
	o.setDefault("window_fullscreen", "false")
	o.setDefault("window_maximized", "false")

	if WINDOWS == runtime.GOOS {
		o.setDefault("update_winapp2", "false")
		o.setDefault("win10_theme", "false")
	}

	if _, err := o.config.GetSection("preserve_languages"); err != nil {
		lang := userLocale
		pos := strings.Index(lang, "_")
		if -1 != pos {
			lang = lang[0:pos]
		}

		for _, lang_ := range []string{lang, "en"} {
			log.Printf("Automatically preserving language %s.\n", lang_)
			o.setLanguage(lang_, true)
		}
	}

	// BleachBit upgrade or first start ever
	if !o.config.Section("bleachbit").HasKey("version") || o.get("version", "bleachbit") != APP_VERSION {
		o.set("first_start", "true", "", true)
	}

	// set version
	o.set("version", APP_VERSION, "", true)
}

func (o *options) get(option, section string) bool {
	if section == "" {
		section = "bleachbit"
	}

	if WINDOWS != runtime.GOOS && "update_winapp2" == option {
		return false
	}

	if "hashpath" == section && option[1] == ':' {
		option = string(option[0]) + option[2:]
	}

	optionInBoolKeys := valueInList(option, &booleanKeys)
	optionInIntKeys := valueInList(option, &intKeys)

	if optionInBoolKeys {
		if "bleachbit" == section && "debug" == option && isDebuggingEnabledViaCli() {
			return true
		}

		key, err := o.config.Section(section).GetKey(option)
		if err != nil {
			log.Fatalln(err.Error())
		}
		b, err := key.Bool()
		if err != nil {
			log.Fatalln(err.Error())
		}
		return b
	} else if optionInIntKeys {
		key, err := o.config.Section(section).GetKey(option)
		if err != nil {
			log.Fatalln(err.Error())
		}
		intVal, err := key.Int()
		if err != nil {
			log.Fatalln(err.Error())
		}

		if intVal != 0 {
			return true
		} else {
			return false
		}
	}

	return
}

func (o *options) setHashPath(pathname, hashValue string) {
	o.set(pathToOption(pathname), hashValue, "hashpath", true)
}

func (o *options) setLanguage(langID string, value bool) {
	name := "preserve_languages"
	langSec := o.config.Section(name)
	if langSec.HasKey(name) && !value {
		langSec.DeleteKey(name)
	} else {
		_, err := langSec.NewKey(name, strconv.FormatBool(value))
		if err != nil {
			log.Fatalln(err.Error())
		}

		o.flush()
	}
}

func (o *options) setDefaultBool(key string, value bool) {

}

func (o *options) setList(key string, values []string) error {
	section := fmt.Sprintf("list/%s", key)

	// delete section if it exist
	if _, err := o.config.GetSection(section); err == nil {
		o.config.DeleteSection(section)
	}

	newSection := o.config.Section(section)

	for i, value := range values {
		_, err := newSection.NewKey(strconv.Itoa(i), value)
		if err != nil {
			return err
		}
	}

	defer o.flush()
	return nil
}
