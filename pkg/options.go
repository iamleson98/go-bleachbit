package pkg

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	_ "github.com/leminhson2398/bleachbit/log"
	log "github.com/sirupsen/logrus"
	ini "gopkg.in/ini.v1"
)

var (
	booleanKeys []string
	intKeys     []string
	options_    *options

	// runtime type checking
	_ optionsInterface = newOptions()
)

type GetType string

const (
	getString GetType = "string"
	getBool   GetType = "boolean"
	getInt    GetType = "integer"
	getFloat  GetType = "float"
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
		// FIXME
		log.WithField("spot", "options.pathToOption()").Fatalln("not implemented")
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
		log.WithFields(log.Fields{
			"spot": "options.initConfiguration",
		}).Infof("Deleting configuration: %s", optionsFile)

		err := os.Remove(optionsFile)

		log.WithFields(log.Fields{
			"spot": "options.initConfiguration",
		}).Warningln(err.Error())
	}

	fIni, err := os.Create(optionsFile)
	if err != nil {
		log.WithField("spot", "options.initConfiguration").Fatalln(err.Error())
	}
	defer fIni.Close()

	_, err = fIni.WriteString("[bleachbit]\n")
	if err != nil {
		log.WithField("spot", "options.initConfiguration").Fatalln(err.Error())
	}
	if WINDOWS == runtime.GOOS && portableMode {
		fIni.WriteString("[Portable]\n")
	}

	for _, section := range options_.config.Sections() {
		options_.config.DeleteSection(section.Name())
	}

	options_.restore()
}

// optionsInterface helps check whether a type is properly implemented or not
type optionsInterface interface {
	flush()
	purge()
	setDefault(key, value string)
	hasOption(option, section string) bool
	get(option, section string, getType GetType) interface{}
	getHashPath(pathname string) interface{}
	getLanguage(langID string) bool
	getLanguages() []string
	getList(option string) []string
	getPaths(section string) [][2]string
	getWhitelistPaths() [][2]string
	getCustomPaths() [][2]string
	getTree(parent, child string) bool
	isCorrupt() bool
	restore()
	set(key, value, section string, commit bool)
	commit()
	setHashpath(pathname, hashvalue string)
	setList(key string, values []string) error
	setWhitelistPaths(values []string)
	setCustomPaths(values [][2]string)
	setLanguage(langid string, value bool)
	setTree(parent, child, value string)
	toggle(key string)
}

type options struct {
	purged bool
	config *ini.File
}

func newOptions() *options {
	opts := new(options)
	opts.purged = false

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
		log.WithField("spot", "options.options.flush").Fatalln(err.Error())
	}

	if mkFile && sudoMode() {
		chownself(optionsFile)
	}
}

// purge clears out obsolete data
func (o *options) purge() {
	o.purged = true
	if hashPathSec, err := o.config.GetSection("hashpath"); err != nil {
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
				log.WithField("spot", "options.options.purge()").Infoln("Error checking whether [ath exists")
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
		log.WithField("spot", "options.options.setDefault()").Fatalln(err.Error())
	}
}

func (o *options) hasOption(option, section string) bool {
	if section == "" {
		section = "bleachbit"
	}

	return o.config.Section(section).HasKey(option)
}

// get retrieves option from given section
// returned value will be string and the caler must parse the returned values itself
func (o *options) get(option, section string, getType GetType) interface{} {
	if section == "" {
		section = "bleachbit"
	}

	if WINDOWS != runtime.GOOS && "update_winapp2" == option {
		return false
	}

	if "hashpath" == section && string(option[1]) == ":" {
		option = string(option[0]) + option[2:]
	}

	// get section
	key, err := o.config.Section(section).GetKey(option)
	if err != nil {
		log.WithField("spot", "options.options.get()").Fatalln(err.Error())
	}

	if valueInList(option, &booleanKeys) {
		if "bleachbit" == section && "debug" == option && isDebuggingEnabledViaCli() {
			return true
		}
		return parseOption(key, getType)

	} else if valueInList(option, &intKeys) {
		return parseOption(key, getType)
	}
	return parseOption(key, getType)
}

func parseOption(key *ini.Key, getType GetType) interface{} {
	var res interface{}

	switch getType {
	case getBool:
		res = key.MustBool()
	case getFloat:
		res = key.MustFloat64()
	case getInt:
		res = key.MustInt()
	case getString:
		res = key.String()
	}

	return res
}

func (o *options) getHashPath(pathname string) interface{} {
	return o.get(pathToOption(pathname), "bleachbit", getString)
}

// getLanguage retrieves value for whether to preserve the language
func (o *options) getLanguage(langID string) bool {
	if !o.config.Section("preserve_languages").HasKey(langID) {
		return false
	}

	key, err := o.config.Section("preserve_languages").GetKey(langID)
	if err != nil {
		return false
	}

	return key.MustBool()
}

func (o *options) getLanguages() []string {
	if sec, err := o.config.GetSection("preserve_languages"); err != nil {
		return []string{}
	} else {
		keys := sec.Keys()
		keyNames := make([]string, len(keys))
		for _, key := range keys {
			keyNames = append(keyNames, key.Name())
		}
		return keyNames
	}
}

func (o *options) getList(option string) []string {
	section := "list/" + option
	if _, err := o.config.GetSection(section); err != nil {
		return []string{}
	}

	keys := o.config.Section(section).Keys()
	values := make([]string, len(keys))

	// increment sort al the key names
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Name() < keys[j].Name()
	})

	for _, key := range keys {
		values = append(values, key.Value())
	}

	return values
}

func (o *options) getPaths(section string) [][2]string {
	if sec, err := o.config.GetSection(section); err != nil {
		return [][2]string{}
	} else {
		keys := sec.Keys()

		// incrementing sort keys by their names
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Name() < keys[j].Name()
		})

		meetMap := make(map[string]bool)
		values := [][2]string{}
		for _, key := range keys {
			pos := strings.Index(key.Name(), "_")
			if -1 == pos {
				continue
			}

			newKeyName := key.Value()[0:pos]
			if _, ok := meetMap[newKeyName]; !ok {
				meetMap[newKeyName] = true
				values = append(values, [2]string{
					sec.Key(newKeyName + "_type").Value(),
					sec.Key(newKeyName + "_path").Value(),
				})
			}
		}

		return values
	}
}

func (o *options) getWhitelistPaths() [][2]string {
	return o.getPaths("whitelist/paths")
}

// getTree Retrieve an option for the tree view.  The child may be emtpty string
func (o *options) getTree(parent, child string) bool {
	option := parent
	if child != "" {
		option += "." + child
	}

	if !o.config.Section("tree").HasKey(option) {
		return false
	}

	key := o.config.Section("tree").Key(option)
	boolVal, err := key.Bool()
	if err != nil {
		log.WithField("spot", "options.options.getTree()").Errorln("Error in getTree(): " + err.Error())
		return false
	}
	return boolVal
}

// isCorrupt Perform a self-check for corruption of the configuration
func (o *options) isCorrupt() bool {
	for _, key := range booleanKeys {
		if o.config.Section("bleachbit").HasKey(key) {
			k := o.config.Section("bleachbit").Key(key)
			_, err := k.Bool()
			if err != nil {
				return true
			}
		}
	}

	for _, intKey := range intKeys {
		if o.config.Section("bleachbit").HasKey(intKey) {
			key := o.config.Section("bleachbit").Key(intKey)
			_, err := key.Int()
			if err != nil {
				return true
			}
		}
	}

	return false
}

func (o *options) set(key, value, section string, commit bool) {
	if section == "" {
		section = "bleachbit"
	}
	sec := o.config.Section(section)
	if k, err := sec.GetKey(key); err != nil {
		_, err := sec.NewKey(key, value)
		if err != nil {
			log.WithField("spot", "options.options.set()").Fatalln(err.Error())
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
		log.WithField("spot", "options.options.restore()").Errorf("Error reading application's configuration %s\n", err.Error())
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
			log.WithField("spot", "options.options.restore()").Errorln(err.Error())
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
			log.WithField("spot", "options.options.setDefault()").Infof("Automatically preserving language %s.\n", lang_)
			o.setLanguage(lang_, true)
		}
	}

	// BleachBit upgrade or first start ever
	if !o.config.Section("bleachbit").HasKey("version") || o.get("version", "bleachbit", getString) != APP_VERSION {
		o.set("first_start", "true", "", true)
	}

	// set version
	o.set("version", APP_VERSION, "", true)
}

// setHashPath Remember the hash of a path
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
			log.WithField("spot", "options.options.setLanguage()").Fatalln(err.Error())
		}

		o.flush()
	}
}

// setList Set a value which is a list data type
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

// setWhitelistPaths saves with whitelist
func (o *options) setWhitelistPaths(values []string) {
	section := "whitelist/paths"
	if _, err := o.config.GetSection(section); err == nil {
		o.config.DeleteSection(section)
	}

	sec := o.config.Section(section)

	for i, val := range values {
		_, err := sec.NewKey(strconv.Itoa(i)+"_type", string(val[0]))
		if err != nil {
			log.WithField("spot", "options.options.setWhitelistPaths()").Errorln(err.Error())
		}
	}

	o.flush()
}

func (o *options) setTree(parent, child, value string) {
	var treeSec *ini.Section
	if _, err := o.config.GetSection("tree"); err != nil {
		treeSec = o.config.Section("tree")
	}

	option := parent

	if child != "" {
		option = option + "." + child
	}
	if treeSec.HasKey(option) && value == "" {
		treeSec.DeleteKey(option)
	} else {
		_, err := treeSec.NewKey(option, value)
		if err != nil {
			log.WithField("spot", "options.options.setTree()").Errorln(err.Error())
		}
	}

	o.flush()
}

// toggle toggles a boolean key
func (o *options) toggle(key string) {
	valAtKey := o.get(key, "bleachbit", getBool)
	value := "false"
	if valAtKey == true {
		value = "true"
	}
	o.set(key, value, "bleachbit", true)
}

func (o *options) getCustomPaths() [][2]string {
	return o.getPaths("custom/paths")
}

// setCustomPaths saves the customlist
func (o *options) setCustomPaths(values [][2]string) {
	section := "custom/paths"
	if sec, err := o.config.GetSection(section); err != nil {
		o.config.DeleteSection(section)
	} else {
		for i, v := range values {
			_, err := sec.NewKey(strconv.Itoa(i)+"_type", v[0])
			if err != nil {
				log.WithField("spot", "options.options.setCustomPaths()").Fatalln(err.Error())
			}
			_, err = sec.NewKey(strconv.Itoa(i)+"_path", v[1])
			if err != nil {
				log.WithField("spot", "options.options.setCustomPaths()").Fatalln(err.Error())
			}
		}
		o.flush()
	}
}

func (o *options) setHashpath(pathname, hashvalue string) {
	o.set(pathToOption(pathname), hashvalue, "bleachbit", true)
}
