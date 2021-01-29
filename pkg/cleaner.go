package pkg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

const (
	// LINUX if os is linux
	LINUX string = "linux"

	// WINDOWS if os is windows
	WINDOWS string = "windows"

	// DARWIN
	DARWIN string = "darwin"
)

// Cleaner represents a general cleaner.
// Other cusom-cleaner (system junk, internet, browser, ...) inherit from this base
type Cleaner struct {
	actions         [][2]string
	id              string
	description     string
	name            string
	options         map[string][2]string
	running         [][2]string
	warnings        map[string]string
	regexesCompiled []*regexp.Regexp
}

func (c *Cleaner) addAction(optionID, action string) {
	c.actions = append(c.actions, [2]string{optionID, action})
}

func (c *Cleaner) addOption(optionID, name, description string) {
	c.options[optionID] = [2]string{name, description}
}

func (c *Cleaner) addRunning(detectionType, pathname string) {
	c.running = append(c.running, [2]string{detectionType, pathname})
}

func (c *Cleaner) autoHide() bool {

	// for _, option := range c.getOptions() {
	// 	_ := option[0]

	// }

	// return true

	panic("not implemented")
}

func (c *Cleaner) getCommands(optionID string) interface{} {
	panic("not implemented")
	// for _, action := range c.actions {
	// 	if optionID == action[0] {

	// 	}
	// }

	// if _, ok := c.options[optionID]; !ok {
	// 	panic(fmt.Sprintf("unknown option %q", optionID))
	// }
}

func (c *Cleaner) getDeepScan(optionID string) {
	panic("not implemented")
}

func (c *Cleaner) getDescription() string {
	return c.description
}

func (c *Cleaner) getID() string {
	return c.id
}

func (c *Cleaner) getName() string {
	return c.name
}

func (c *Cleaner) getOptionDescriptions() [][2]string {
	optionKeys := make([]string, len(c.options))
	results := make([][2]string, len(c.options))

	for key := range c.options {
		optionKeys = append(optionKeys, key)
	}

	incrementSort(&optionKeys)

	for _, key := range optionKeys {
		results = append(results, c.options[key])
	}

	return results
}

func (c *Cleaner) getOptions() [][2]string {
	results := [][2]string{}
	optionKeys := make([]string, len(c.options))

	for key := range c.options {
		optionKeys = append(optionKeys, key)
	}

	incrementSort(&optionKeys)

	for _, key := range optionKeys {
		results = append(results, [2]string{key, c.options[key][0]})
	}

	return results
}

func (c *Cleaner) getWarning(optionID string) *string {
	if value, in := c.warnings[optionID]; in {
		return &value
	}
	return nil
}

func (c *Cleaner) isRunning() bool {
	for _, running := range c.running {

		test := running[0]
		pathname := running[1]

		if "exe" == test {
			if LINUX == runtime.GOOS && isRunning(pathname) {
				log.Printf("process '%s' is running\n", pathname)
				return true
			} else if WINDOWS == runtime.GOOS {
				// TODO: add GOOS == windows
				panic("not implemented")
			}
		} else if "pathname" == test {
			expandVars := os.ExpandEnv(pathname)
			expanded := ExpandUser(expandVars)

			matches, err := filepath.Glob(expanded)
			if err != nil {
				panic(err.Error())
			}

			for _, globbed := range matches {
				_, err := os.Stat(globbed)
				if os.IsExist(err) {
					log.Printf("file %s exists indicating %s is running\n", globbed, c.name)
					return true
				}
			}
		} else {
			panic(fmt.Sprintf("unknown running-detection test %q", test))
		}
	}

	return false
}

func (c *Cleaner) isUsable() bool {
	return len(c.actions) > 0
}

func (c *Cleaner) setWarning(optionID, description string) {
	c.warnings[optionID] = description
}
