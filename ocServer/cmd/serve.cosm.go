//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
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

type CosmidManifestEntry struct {
	CID      string `json:"cosmid"`
	Name     string `json:"name"`
	IsPublic bool   `json:"is_public"`
}
type CosmidManifestResponse struct {
	Count int                   `json:"count"`
	Data  []CosmidManifestEntry `json:"data"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
func HandlerCosmManifest(httpResponse http.ResponseWriter, r *http.Request) {

	var manifestResponse CosmidManifestResponse
	manifestResponse.Count = CurrentJamManifest.NumberOfCOSMIDs()
	manifestResponse.Data = make([]CosmidManifestEntry, 0, manifestResponse.Count)

	SysLog.Info("Manifest requested", zap.String("RemoteAddr", r.RemoteAddr))

	for i := 1; i <= manifestResponse.Count; i++ {
		cosmid := fmt.Sprintf("jam_%03d", i)
		cosmidName, ok := CurrentJamManifest.NameFromCOSMID(cosmid)
		if !ok {
			break
		}
		cosmidPublic, ok := CurrentJamManifest.COSMIDJamIsPublic(cosmid)
		if !ok {
			break
		}

		manifestResponse.Data = append(manifestResponse.Data, CosmidManifestEntry{cosmid, cosmidName, cosmidPublic})
	}

	handlerEmitJson(httpResponse, manifestResponse)
}
