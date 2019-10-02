package utils

import (
	"regexp"
	"strings"
)

type ContentTypeFilter interface {
	IsAllowed(t string) bool
}

type contentTypeFilter struct {
	filterPattern string
}

// Retrieves a new content filter. If the lis is empty all content types will be allowed
func NewContentTypeFilter(types []string) ContentTypeFilter {
	if len(types) == 0 {
		return &contentTypeFilter{filterPattern: ".*"}
	}
	pattern := "(" + strings.Join(types, "?)|(") + "?)"
	return &contentTypeFilter{filterPattern: pattern}
}

func (f *contentTypeFilter) IsAllowed(t string) bool {
	r := regexp.MustCompile(f.filterPattern)
	return r.MatchString(t)
}
