//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"net/http"
	"sync/atomic"
	"time"
)

type StatusResponse struct {
	Version                   int   `json:"version"`                   // everything needs a version number
	Awake                     bool  `json:"awake"`                     // awake! eventually this could return false and the client could deal with that
	ServerTime                int64 `json:"serverTime"`                // what time it is on the server
	MostRecentPublicJamChange int64 `json:"mostRecentPublicJamChange"` // the unix timestamp of the most recently published public riff
}

// -----------------------------------------------------------------------------------------------------------------------------------
// an endpoint used by the cosm client to check if a server is alive, what it thinks the time is, things of that nature
func HandlerCosmStatus(httpResponse http.ResponseWriter, r *http.Request) {

	latestPublicTimestamp := atomic.LoadInt64(&publicJamsLastChangeTimestamp)

	statusResponse := StatusResponse{
		1,
		true,
		time.Now().UnixMilli(),
		latestPublicTimestamp,
	}

	handlerEmitJson(httpResponse, statusResponse)
}
