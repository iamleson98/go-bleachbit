package pkg

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	ENCODING "github.com/leminhson2398/bleachbit/encoding"
)

var (
	passwdFile  string
	passwdPath  []string
	fieldSep    map[string]pathConv
	errNoPassDB = errors.New("no password database")
	envVars     = []string{"LC_ALL", "LC_CTYPE", "LANG", "LANGUAGE"}
)

func init() {
	passwdPath = []string{}

	if etcPasswd := os.Getenv("ETC_PASSWD"); etcPasswd != "" {
		passwdPath = append(passwdPath, "ETC_PASSWD")
	}
	if etc := os.Getenv("ETC"); etc != "" {
		passwdPath = append(passwdPath, fmt.Sprintf("%s/passwd", etc))
	}
	// NOTE: this is borrowed from bleachbit python
	if pythonHome := os.Getenv("PYTHONHOME"); pythonHome != "" {
		passwdPath = append(passwdPath, fmt.Sprintf("%s/Etc/passwd", pythonHome))
	}

	// assign the first file exists to passwdFile
	for _, v := range passwdPath {
		file, err := os.Open(v)
		defer file.Close()
		if err != nil || err == os.ErrNotExist {
			continue
		}

		passwdFile = v
		break
	}

	fieldSep = map[string]pathConv{
		":": pathConv(unixPathConv),
	}
	if os.PathSeparator != ':' {
		fieldSep[string(os.PathSeparator)] = pathConv(nullPathConv)
	}

	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type Passwd struct {
	pwName   string
	pwPasswd string
	pwUid    int
	pwGid    int
	pwGecos  string
	pwDir    string
	pwShell  string
}

type pathConv func(path string) string

func unixPathConv(path string) string {
	var conv string

	if string(path[0]) == "$" {
		conv = string(path[1]) + ":" + string(path[2:])
	} else if string(path[1]) == ";" {
		conv = string(path[0]) + ":" + string(path[2:])
	} else {
		conv = path
	}

	// NOTE: in python it should be conv.replace(os.altsep, os.sep)
	return nullPathConv(conv)
}

func nullPathConv(path string) string {
	return strings.ReplaceAll(path, "\\", string(os.PathSeparator))
}

// incrementSort takes a pointer to a slice of string and sort inner items without returns
func incrementSort(arr *[]string) {
	swapped := true
	for swapped {
		swapped = false
		for i := 0; i < len(*arr)-1; i++ {
			if (*arr)[i] > (*arr)[i+1] {
				(*arr)[i], (*arr)[i+1] = (*arr)[i+1], (*arr)[i]
				swapped = true
			}
		}
	}
}

func getPwUID(uid int) (*Passwd, error) {
	u, _, err := readPasswordFile()
	if err != nil {
		return &Passwd{}, err
	}
	return u[uid], nil
}

func getFieldSep(record string) string {
	var fs string

	for key := range fieldSep {
		if strings.Count(record, key) == 6 {
			fs = key
			break
		}
	}

	if fs != "" {
		return fs
	}
	panic("passwd database fields not delimited")
}

func readPasswordFile() (map[int]*Passwd, map[string]*Passwd, error) {

	uidx := make(map[int]*Passwd)
	namx := make(map[string]*Passwd)

	if passwdFile != "" {

		var sep string = ""

		data, err := ioutil.ReadFile(passwdFile)
		if err != nil {
			return uidx, namx, err
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {

			line := scanner.Text()
			line = strings.TrimSpace(line)

			if len(line) > 6 {
				if sep == "" {
					sep = getFieldSep(line)
				}
				fields := strings.Split(line, " ")
				intFields := make([]int, len(fields))

				for _, v := range []int{2, 3} {
					intVal, err := strconv.Atoi(fields[v])
					if err != nil {
						// panic(fmt.Sprintf("error parsing value: %v", err.Error()))
						return uidx, namx, err
					}
					intFields[v] = intVal
				}

				for _, v := range []int{5, 6} {
					fields[v] = fieldSep[sep](fields[v])
				}

				record := Passwd{
					pwName:   fields[0],
					pwPasswd: fields[1],
					pwUid:    intFields[2],
					pwGid:    intFields[3],
					pwGecos:  fields[4],
					pwDir:    fields[5],
					pwShell:  fields[6],
				}

				if _, ok := uidx[intFields[2]]; !ok {
					uidx[intFields[2]] = &record
				}
				if _, ok := namx[fields[0]]; !ok {
					namx[fields[0]] = &record
				}
			} else if len(line) > 0 {
				continue
			} else {
				break
			}
		}

		if len(uidx) == 0 {
			return uidx, namx, errors.New("length is 0")
		}
		return uidx, namx, nil
	}

	return uidx, namx, errNoPassDB
}

func getPwNam(name string) (*Passwd, error) {
	_, n, err := readPasswordFile()
	if err == nil {
		return &Passwd{}, err
	}

	return n[name], nil
}

// ExpandUser is borrowed from python code
// refer https://github.com/python/cpython/blob/bf64d9064ab641b1ef9a0c4bda097ebf1204faf4/Lib/posixpath.py#L228
func ExpandUser(path string) string {

	tilde := "~"
	if !strings.HasPrefix(path, tilde) {
		return path
	}

	sep := "/"

	i := 1
	for ; i < len(path); i++ {
		if string(path[i]) == sep {
			break
		}
	}

	if i < 0 {
		i = len(path)
	}

	var userHome string

	if i == 1 {
		homeEnv := os.Getenv("HOME")
		if homeEnv == "" { // "HOME" not in environment variable list
			pwUID, err := getPwUID(os.Getuid())
			if err != nil {
				return path
			}
			userHome = pwUID.pwDir
		} else {
			userHome = homeEnv
		}
	} else {
		var name string
		// go slicing is difference from python's
		if i < 1 {
			name = ""
		} else {
			name = path[1:i]
		}
		pwent, err := getPwNam(name)
		fmt.Println("pwent is:", pwent)
		if err != nil {
			return path
		}
		userHome = pwent.pwDir
	}

	// if no user home, return the path unchanged on VxWorks
	if userHome == "" && runtime.GOOS == "vxworks" {
		return path
	}

	root := sep
	userHome = strings.TrimRight(userHome, root)
	if result := userHome + path[i:]; result != "" {
		return result
	}
	return root
}

func itemExist(pathname string) bool {
	_, err := os.Stat(pathname)
	if err == nil {
		return true
	}
	return false
}

func lExists(path string) bool {
	_, err := os.Lstat(path)
	if err != nil {
		return false
	}

	return true
}

func getLogin() (string, error) {
	envNames := []string{"LOGNAME", "USER", "LNAME", "USERNAME"}
	for _, v := range envNames {
		variable := os.Getenv(v)
		if variable != "" {
			return variable, nil
		}
	}

	pass, err := getPwUID(os.Getuid())
	if err != nil {
		// log.Fatalf("Error: %v\n", err)
		return "", err
	}

	return pass.pwName, nil
}

func product(params ...[]string) chan []string {
	c := make(chan []string)
	var wg sync.WaitGroup
	wg.Add(1)

	iterate(&wg, c, []string{}, params...)

	go func() {
		wg.Wait()
		close(c)
	}()

	return c
}

func iterate(wg *sync.WaitGroup, channel chan []string, result []string, params ...[]string) {
	defer wg.Done()
	if len(params) == 0 {
		channel <- result
		return
	}

	p, params := params[0], params[1:]

	for i := 0; i < len(p); i++ {
		wg.Add(1)
		resultCopy := append([]string{}, result...)
		go iterate(wg, channel, append(resultCopy, p[i]), params...)
	}
}

func zip(keys []string, vals []string) map[string]string {
	if len(keys) != len(vals) {
		panic("keys and vals must be equal in length")
	}

	m := make(map[string]string, len(keys))
	for i := 0; i < len(keys); i++ {
		m[keys[i]] = vals[i]
	}

	return m
}

func getDefaultLocale() (string, string) {
	var localeName string
	for _, variable := range envVars {
		localeName = os.Getenv(variable)
		if len(localeName) > 0 {
			if variable == "LANGUAGE" {
				localeName = strings.Split(localeName, ":")[0]
			}
			break
		}
	}

	if localeName == "" {
		localeName = "C"
	}

	return parseLocaleName(localeName)
}

func parseLocaleName(localeName string) (string, string) {
	code := normalize(localeName)
	if strings.Index(code, "@") > -1 {
		splitCode := strings.SplitN(code, "@", 1)
		code = splitCode[0]
		modifier := splitCode[1]
		if modifier == "euro" && strings.Index(code, ".") == -1 {
			return code, "iso-8859-15"
		}
	}

	if strings.Index(code, ".") > -1 {
		splitCode := strings.Split(code, ".")[0:2]
		return splitCode[0], splitCode[1]
	} else if code == "C" {
		return "", ""
	} else if code == "UTF-8" {
		return "", "UTF-8"
	}

	return "", ""
}

func normalize(localeName string) string {
	code := strings.ToLower(localeName)
	if strings.Index(code, ":") > -1 {
		code = strings.ReplaceAll(code, ":", ".")
	}

	var modifier string

	if strings.Index(code, "@") > -1 {
		splitCode := strings.SplitN(code, "@", 1)
		code = splitCode[0]
		modifier = splitCode[1]
	}

	var langName, encoding string
	if strings.Index(code, ".") > -1 {
		splitCode := strings.Split(code, ".")[0:2]
		langName = splitCode[0]
		encoding = splitCode[1]
	} else {
		langName = code
	}

	langEnc := langName
	if encoding != "" {
		normEncoding := strings.ReplaceAll(encoding, "-", "")
		normEncoding = strings.ReplaceAll(normEncoding, "_", "")
		langEnc = langEnc + "." + normEncoding
	}
	lookupName := langEnc
	if modifier != "" {
		lookupName = lookupName + "@" + modifier
	}
	if val, ok := ENCODING.LocaleAlias[lookupName]; ok {
		code = val
		return code
	}

	if modifier != "" {
		if val, ok := ENCODING.LocaleAlias[langEnc]; ok {
			code = val
			if strings.Index(code, "@") == -1 {
				return appendModifier(code, modifier)
			}

			splitCode := strings.SplitN(code, "@", 1)[1]
			if strings.ToLower(splitCode) == modifier {
				return code
			}
		}
	}

	if encoding != "" {
		lookupName = langName
		if modifier != "" {
			lookupName = lookupName + "@" + modifier
		}
		if val, ok := ENCODING.LocaleAlias[lookupName]; ok {
			code = val
			if strings.Index(code, "@") == -1 {
				return replaceEncoding(code, encoding)
			}
			splitCode := strings.SplitN(code, "@", 1)
			code = splitCode[0]
			modifier = splitCode[1]
			return replaceEncoding(code, encoding) + "@" + modifier
		}

		if modifier != "" {
			if val, ok := ENCODING.LocaleAlias[langName]; ok {
				code = val
				if strings.Index(code, "@") == -1 {
					code = replaceEncoding(code, encoding)
					return appendModifier(code, modifier)
				}
				splitCode := strings.SplitN(code, "@", 1)
				code = splitCode[0]
				defmod := splitCode[1]
				if strings.ToLower(defmod) == modifier {
					return replaceEncoding(code, encoding) + "@" + defmod
				}
			}
		}
	}

	return localeName
}

func replaceEncoding(code, encoding string) string {
	var langName string
	if strings.Index(code, ".") > -1 {
		langName = code[0:strings.Index(code, ".")]
	} else {
		langName = code
	}

	normEncoding := normalizeEncoding(encoding)
	if val, ok := ENCODING.Aliases[strings.ToLower(normEncoding)]; ok {
		normEncoding = val
	}

	encoding = normEncoding
	normEncoding = strings.ToLower(normEncoding)
	if val, ok := ENCODING.LocaleEncodingAlias[normEncoding]; ok {
		encoding = val
	} else {
		normEncoding = strings.ReplaceAll(normEncoding, "_", "")
		normEncoding = strings.ReplaceAll(normEncoding, "-", "")
		if val, ok := ENCODING.LocaleEncodingAlias[normEncoding]; ok {
			encoding = val
		}
	}

	return langName + "." + encoding
}

func appendModifier(code, modifier string) string {
	if modifier == "euro" {
		if strings.Index(code, ".") == -1 {
			return code + ".ISO8859-15"
		}
		encoding := partition(code, ".")[2]
		if encoding == "ISO8859-15" || encoding == "UTF-8" {
			return code
		}
		if encoding == "ISO8859-1" {
			return replaceEncoding(code, "ISO8859-15")
		}
	}

	return code + "@" + modifier
}

func normalizeEncoding(encoding string) string {
	chars := []string{}
	punct := false
	for _, c := range encoding {
		if isAlNum(c) || c == '.' {
			if punct && len(chars) > 0 {
				chars = append(chars, "_")
			}
			chars = append(chars, string(c))
			punct = true
		} else {
			punct = true
		}
	}

	return strings.Join(chars, "")
}
