package pkg

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	ini "gopkg.in/ini.v1"
)

var (
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
)

func init() {
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
	file, err := os.Create(optionsFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer file.Close()

	_, err = o.config.WriteTo(file)
	if err != nil {
		log.Fatalln(err.Error())
	}

	if mkFile && sudoMode() {
		chownself(optionsFile)
	}
}

func (o *options) purge() {
	o.purged = true
	hashPathSec, err := o.config.GetSection("hashpath")
	if err != nil {
		return
	}

	hashPathSec.ChildSections()
}

func (o *options) setDefault(key, value string, isBool bool) {
	section := o.config.Section("bleachbit")
	if section == nil {
		log.Fatalln("no section named 'bleachbit'")
	}

	o.set(key, value, "", isBool, true)
}

func (o *options) set(key, value, section string, isBool, commit bool) {
	if section == "" {
		section = "bleachbit"
	}
	sec := o.config.Section(section)
	if sec == nil {
		log.Fatalf("No section named %s", section)
	}

	if isBool {
		key, err := sec.NewBooleanKey(key)
		if err != nil {
			log.Fatalln(err.Error())
		}
		key.SetValue(value)
	} else {
		sec.Key(key).SetValue(value)
	}

	if commit {
		o.flush()
	}
}

func (o *options) commit() {
	o.flush()
}

func (o *options) restore() {
	cfg, err := ini.Load(optionsFile)
	if err != nil {
		log.Printf("Error reading application's configuration %s\n", err.Error())
	}

	o.config = cfg

	if _, err := o.config.GetSection("bleachbit"); err != nil {
		// NewSection() returns a new section and error, but error only matter if section name provided is empty string.
		// we can ignore them
		o.config.NewSection("bleachbit")
	}
	if _, err := o.config.GetSection("hashpath"); err != nil {
		o.config.NewSection("hashpath")
	}
	if _, err := o.config.GetSection("list/shred_drives"); err != nil {
		guessOvrPaths := guessOverritePaths()
		err := o.setList("shred_drives", guessOvrPaths)
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

	// set defaults
	o.setDefault("auto_hide", "true", true)
	o.setDefault("check_beta", "false", true)
	o.setDefault("check_online_updates", "true", true)
	o.setDefault("dark_mode", "true", true)
	o.setDefault("delete_confirmation", "true", true)
	o.setDefault("debug", "false", true)
	o.setDefault("exit_done", "false", true)
	o.setDefault("shred", "false", true)
	o.setDefault("units_iec", "false", true)
	o.setDefault("window_fullscreen", "false", true)
	o.setDefault("window_maximized", "false", true)

	if WINDOWS == runtime.GOOS {
		o.setDefault("update_winapp2", "false", true)
		o.setDefault("win10_theme", "false", true)
	}

	if _, err := o.config.GetSection("preserve_languages"); err != nil {

	}
}

func (o *options) setDefaultBool(key string, value bool) {

}

func (o *options) setList(key string, values []string) error {
	section := fmt.Sprintf("list/%s", key)
	if _, err := o.config.GetSection(section); err == nil {
		o.config.DeleteSection(section)
	}

	newSection, err := o.config.NewSection(section)
	if err != nil {
		return err
	}
	counter := 0
	for _, value := range values {
		_, err := newSection.NewKey(strconv.Itoa(counter), value)
		if err != nil {
			return err
		}
		counter++
	}

	defer o.flush()
	return nil
}
