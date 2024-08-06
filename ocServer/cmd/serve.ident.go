//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"net/http"
	"unicode"

	"github.com/spf13/viper"
)

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

type ServerIdent struct {
	fourcc string
}

func NewServerIdent() *ServerIdent {
	h := &ServerIdent{}
	h.fourcc = "ishani:ourocosm:" + viper.GetString(cConfigCosmFourCC)
	return h
}

func (h *ServerIdent) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	headers := w.Header()
	headers.Add("Server", h.fourcc)

	next(w, r)
}
