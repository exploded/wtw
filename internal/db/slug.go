package db

import (
	"regexp"
	"strings"
)

var (
	nonAlphaNum    = regexp.MustCompile(`[^a-z0-9-]`)
	multipleHyphen = regexp.MustCompile(`-{2,}`)
)

func Slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphaNum.ReplaceAllString(s, "")
	s = multipleHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
