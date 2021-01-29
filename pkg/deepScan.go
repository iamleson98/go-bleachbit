package pkg

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type dirScanResult struct {
	dirPath string
	files   []string
}

type listDirScanResults []*dirScanResult

func (l listDirScanResults) addFile(filePath string) {
	fileDirName := filepath.Dir(filePath)
	for _, result := range l {
		if result.dirPath == fileDirName {
			result.files = append(result.files, filePath)
		}
	}
}

func normalizedWalk(top string) (listDirScanResults, error) {

	var listDirScan listDirScanResults

	err := filepath.Walk(top, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			dsr := new(dirScanResult)
			dsr.dirPath = path
			listDirScan = append(listDirScan, dsr)
		} else {
			// file
			listDirScan.addFile(path)
		}

		return err
	})

	if err != nil {
		return []*dirScanResult{}, err
	}

	return listDirScan, nil
}

type compiledSearchOpts struct {
	regex       string
	nregex      string
	wholeregex  string
	nwholeregex string
	command     string
}

type compiledSearch struct {
	regex       *regexp.Regexp
	nregex      *regexp.Regexp
	wholeregex  *regexp.Regexp
	nwholeregex *regexp.Regexp
	command     string
}

func reCompile(pattern string) *regexp.Regexp {
	rgxp, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	return rgxp
}

func newCompiledSearch(opts compiledSearchOpts) *compiledSearch {

	cs := new(compiledSearch)
	cs.regex = reCompile(opts.regex)
	cs.nregex = reCompile(opts.nregex)
	cs.wholeregex = reCompile(opts.wholeregex)
	cs.nwholeregex = reCompile(opts.nwholeregex)
	cs.command = opts.command

	return cs
}

func (cs *compiledSearch) match(dirpath string, filename string) *string {
	fullPath := filepath.Join(dirpath, filename)

	if cs.regex != nil && !cs.regex.MatchString(filename) {
		return nil
	}

	if cs.nregex != nil && cs.nregex.MatchString(filename) {
		return nil
	}

	if cs.wholeregex != nil && !cs.wholeregex.MatchString(fullPath) {
		return nil
	}

	if cs.nwholeregex != nil && cs.nwholeregex.MatchString(fullPath) {
		return nil
	}

	return &fullPath
}

type deepScan struct {
	searches map[string][]compiledSearchOpts
}

func newDeepScan(searches map[string][]compiledSearchOpts) *deepScan {
	ds := new(deepScan)
	ds.searches = searches

	return ds
}

func (ds *deepScan) scan() {
	bytes, err := json.Marshal(ds.searches)
	if err != nil {
		log.Fatalln(err.Error())
	}
	log.Printf("DeepScan.scan: searches=%s\n", string(bytes))

	// startTime := time.Now().Unix()

	compiledSearches := []*compiledSearch{}
	for key, val := range ds.searches {
		for _, s := range val {
			compiledSearches = append(compiledSearches, newCompiledSearch(s))
		}

		ldsr, err := normalizedWalk(key)
		if err != nil {
			log.Fatalln(err.Error())
		}

		for _, dsr := range ldsr {
			for _, comSearch := range compiledSearches {
				for _, filename := range dsr.files {
					fullName := comSearch.match(dsr.dirPath, filename)

					if fullName != nil {
						if comSearch.command == "delete" {
							panic("not implemented")
						} else if comSearch.command == "shred" {
							panic("not implemented")
						}
					}
				}
			}
		}
	}
}
