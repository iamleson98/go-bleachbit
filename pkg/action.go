package pkg

import (
	"fmt"
	"regexp"
	"strings"
)

type actionProviderInterface interface {
	getDeepScan()
	getCommands()
}

type actionProvider struct {
	plugins []*actionProvider
}

func newActionProvider(actionNode string, pathVars string) *actionProvider {
	ap := new(actionProvider)
	ap.plugins = append(ap.plugins, ap)

	return ap
}

func hasGlob(s string) bool {
	reg := regexp.MustCompile(`[?*\[\]]`)
	return reg.MatchString(s)
}

func (ac *actionProvider) getDeepScan() {
	panic("not implemented")
}

func (ac *actionProvider) getCommands() {
	panic("not implemented")
}

type cache struct {
	seachType string
	path      string
	entries   []string
}

type fileActionProvider struct {
	*actionProvider

	actionKey          string
	cacheableSearchers []string
	cache              cache
}

func newFileActionProvider(actionElement string, pathVars string) *fileActionProvider {
	fap := new(fileActionProvider)

	fap.actionKey = "_file"
	fap.cacheableSearchers = []string{"walk.files"}
	fap.cache = cache{"nothing", "", []string{}}

	return fap
}

func (fap *fileActionProvider) getDeepScan() {

}

func (fap *fileActionProvider) getCommands() {

}

func (fap *fileActionProvider) setPath(rawPath string, pathVars []string) {

}

func expandMultiVar(s string, variables map[string]string) []string {
	if len(variables) == 0 || strings.Index(s, "$$") == -1 {
		return []string{s}
	}

	ret := []string{}
	varsUsed := make(map[string]string)

	for key, val := range variables {
		sub := fmt.Sprintf("$$%s$$", key)
		if strings.Index(s, sub) > -1 {
			varsUsed[key] = val
		}
	}

	if len(varsUsed) == 0 {
		return []string{s}
	}

	varsUsedValues := [][]string{}
	varsUsedKeys := make([]string, len(varsUsed))
	for k, v := range varsUsed {
		splitV := strings.Split(v, "")
		varsUsedValues = append(varsUsedValues, splitV)
		varsUsedKeys = append(varsUsedKeys, k)
	}

	cateProduct := product(varsUsedValues...)

	varsProduct := []map[string]string{}
	for m := range cateProduct {
		zipped := zip(varsUsedKeys, m)
		varsProduct = append(varsProduct, zipped)
	}

	for _, varSet := range varsProduct {
		ms := s
		for k, v := range varSet {
			sub := fmt.Sprintf("$$%s$$", k)
			ms = strings.ReplaceAll(ms, sub, v)
		}
		ret = append(ret, ms)
	}

	if len(ret) > 0 {
		return ret
	}
	return []string{s}
}
