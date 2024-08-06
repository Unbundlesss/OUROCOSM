//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"net/http"
)

// we haven't figured the Join side of it yet
func HandlerJamLeave(httpResponse http.ResponseWriter, r *http.Request) {

	httpResponse.WriteHeader(http.StatusOK)
}
