package kmatch

import "strings"

// KMatcher 用来定义匹配器接口
type KMatcher interface {
	Match(text string) bool
}

type KMatch struct {
	Matches  []string // 至少包含的字符串
	Musts    []string // 必须包含的字符串
	Excludes []string // 需要排除的字符串
}

type kMatcher struct {
	km         KMatch
	ignoreCase bool
}

// NewKMatcher 用来创建一个匹配器
func NewKMatcher(km KMatch, isIgnore bool) KMatcher {
	return &kMatcher{
		ignoreCase: isIgnore,
		km:         km,
	}
}

// Match 用来判断是否匹配
func (m *kMatcher) Match(txt string) bool {
	return m.matchOne(m.km, txt)
}

// 只执行一个match
func (m *kMatcher) matchOne(km KMatch, txt string) bool {
	for _, must := range km.Musts {
		if m.ignoreCase {
			must = strings.ToLower(must)
		}
		if !strings.Contains(txt, must) {
			return false
		}
	}

	for _, exclude := range km.Excludes {
		if m.ignoreCase {
			exclude = strings.ToLower(exclude)
		}
		if strings.Contains(txt, exclude) {
			return false
		}
	}
	im := false
	if m.ignoreCase {
		txt = strings.ToLower(txt)
	}
	for _, match := range km.Matches {
		if m.ignoreCase {
			match = strings.ToLower(match)
		}
		if strings.Contains(txt, match) {
			im = true
			break
		}

	}
	return im
}
