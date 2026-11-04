package tools

import "regexp"

var idPattern = regexp.MustCompile(`^[0-9a-v]{20}$`)

// validID reports whether id is a 20-character NetBird xid.
func validID(id string) bool {
	return idPattern.MatchString(id)
}
