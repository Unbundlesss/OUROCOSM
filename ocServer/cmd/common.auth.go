//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"errors"
	"net/http"
	"strings"
)

// -----------------------------------------------------------------------------------------------------------------------------------
// cut a "Bearer Username:Token" header apart, returning username and token parts
func decodeAccountAuthBearer(r *http.Request) (string, string, error) {
	bearerToken := r.Header.Get("Authorization")
	bearerSplit := strings.Split(bearerToken, "Bearer")
	if len(bearerSplit) != 2 {
		return "", "", errors.New("invalid bearer token")
	}
	bearerComponents := strings.Split(strings.TrimSpace(bearerSplit[1]), ":")
	if len(bearerComponents) != 2 {
		return "", "", errors.New("invalid bearer token contents")
	}

	return bearerComponents[0], bearerComponents[1], nil
}
