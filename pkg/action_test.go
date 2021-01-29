package pkg

import (
	"fmt"
	"testing"
)

func checkMapsSimilar(m1 map[string]string, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v := range m1 {
		if m2v, ok := m2[k]; !ok || (ok && m2v != v) {
			return false
		}
	}

	return true
}

func TestZip(t *testing.T) {
	inputSet1 := [3][]string{
		[]string{"a", "b", "c"},
		[]string{"aab", "bbg", "kijh"},
		[]string{"anh yeu em", "bubuoi", "cconmeMay"},
	}
	inputSet2 := [3][]string{
		[]string{"a", "b", "c"},
		[]string{"aab", "bbg", "kijh"},
		[]string{"anh yeu em", "bubuoi", "cconmeMay"},
	}

	expect1 := map[string]string{"a": "a", "b": "b", "c": "c"}
	expect2 := map[string]string{"aab": "aab", "bbg": "bbg", "c": "c"}
	expect3 := map[string]string{"anh yeu em": "anh yeu em", "bubuoi": "bubuoi", "cconmeMay": "cconmeMay"}

	expectSet := [3]map[string]string{expect1, expect2, expect3}

	for i := 0; i < 3; i++ {
		fmt.Println(i)
		res := zip(inputSet1[i], inputSet2[i])
		if !checkMapsSimilar(res, expectSet[i]) {
			t.Log("Failed")
		} else {
			t.Log("Ok")
		}
	}
}
