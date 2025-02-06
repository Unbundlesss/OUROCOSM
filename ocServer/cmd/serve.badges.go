//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
)

type BadgesQuery struct {
	UserIds []string `json:"userIds"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
func HandlerBadgesQueryPost(httpResponse http.ResponseWriter, r *http.Request) {

	var badgesQuery BadgesQuery
	err := json.NewDecoder(r.Body).Decode(&badgesQuery)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusBadRequest)
		return
	}

	basicResponse := strings.Builder{}

	firstLineEmit := true
	basicResponse.WriteString("{ \"ok\":true, \"data\":{ \"users\": [")
	for _, v := range badgesQuery.UserIds {
		if !firstLineEmit {
			basicResponse.WriteString(", ")
		}
		firstLineEmit = false

		basicResponse.WriteString("{ \"userId\": \"")
		basicResponse.WriteString(v)
		basicResponse.WriteString("\", \"badges\": []}")
	}
	basicResponse.WriteString("] } }")

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	httpResponse.Write([]byte(basicResponse.String()))
}
