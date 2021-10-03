package trinoheaders

import (
	"fmt"
	"net/http"
)

const (
	PreparedStatement    = "Prepared-Statement"
	User                 = "User"
	ClientTags           = "Client-Tags"
	ConnectionProperties = "Connection-Properties"
)

var allowedPrefixes = [...]string{"Presto", "Trino"}

func Get(key string, req *http.Request) string {
	for _, h := range allowedPrefixes {
		s := fmt.Sprintf("X-%s-%s", h, key)

		if val := req.Header.Get(s); val != "" {
			return val
		}
	}
	return ""
}
