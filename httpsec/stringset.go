package httpsec

import (
	"sort"
	"strings"
)

type stringSet map[string]bool

func (s stringSet) add(str string) {
	s[str] = true
}

func (s stringSet) remove(str string) {
	delete(s, str)
}

func (s stringSet) has(str string) bool {
	low := strings.ToLower(str)
	for key := range s {
		if strings.ToLower(key) == low {
			return true
		}
	}
	return false
}

func (s stringSet) slice() []string {
	var strs = make([]string, len(s))
	i := -1
	for str := range s {
		i++
		strs[i] = str
	}
	sort.Strings(strs)
	return strs
}
