package metric

import "strings"

type Labels []string

func (l *Labels) toMapKey() string {
	return strings.Join(*l, "\xff")
}

func (l *Labels) toMap() map[string]string {
	m := make(map[string]string, len(*l))
	for i, lv := range *l {
		m[(*l)[i]] = lv
	}
	return m
}

func WithLabels(val ...string) Labels {
	l := Labels(val)
	return l
}
