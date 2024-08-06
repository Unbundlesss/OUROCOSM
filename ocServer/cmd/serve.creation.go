//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"io"
	"net/http"

	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------------------------------------------------------------
// creating new jams is quite a task; for one, we have no way to spin up a new unique couch id because we don't have the
// short <-> long encryption or hashing method to hand .. so we have to pull from a fixed pile of old ids.
// something we could consider if the project continues long enough, although we would likely need to limit it somehow to
// avoid running out of ids or have this controlled by a ticketing system or something
func HandlerJamCreate(httpResponse http.ResponseWriter, r *http.Request) {

	buf, _ := io.ReadAll(r.Body)
	SysLog.Warn("HandlerJamCreate not handled", zap.String("Body", string(buf)), zap.String("RemoteAddr", r.RemoteAddr))

	httpResponse.WriteHeader(http.StatusUnauthorized)
}
