package trinoheaders

import (
	"fmt"
	"net/http"
)

// https://github.com/trinodb/trino/blob/master/client/trino-client/src/main/java/io/trino/client/ProtocolHeaders.java
// Connection properties is not part of trino client protocol but is sent from some jdbc clients
const (
	PreparedStatement    = "Prepared-Statement"
	User                 = "User"
	ClientTags           = "Client-Tags"
	ConnectionProperties = "Connection-Properties"
	TransactionId        = "Transaction-Id"
	Password             = "Password"
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
