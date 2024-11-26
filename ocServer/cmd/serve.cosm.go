//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	"go.uber.org/zap"
)

type StatusResponse struct {
	Version                       int    `json:"version"`                       // everything needs a version number
	Awake                         bool   `json:"awake"`                         // awake! eventually this could return false and the client could deal with that
	ServerTime                    int64  `json:"serverTime"`                    // what time it is on the server
	MostRecentPublicJamChange     int64  `json:"mostRecentPublicJamChange"`     // the unix timestamp of the most recently published public riff
	MostRecentPublicJamChangeText string `json:"mostRecentPublicJamChangeText"` // Humanize'd version of MostRecentPublicJamChange
	MostRecentPublicJamUser       string `json:"mostRecentPublicJamUser"`       // username from last contribution
	MostRecentPublicJamName       string `json:"mostRecentPublicJamName"`       // name of the jam most recently updated
}

// -----------------------------------------------------------------------------------------------------------------------------------
// an endpoint used by the cosm client to check if a server is alive, what it thinks the time is, things of that nature
func HandlerCosmStatus(httpResponse http.ResponseWriter, r *http.Request) {

	latestJamData := publicJamsLatestData{}
	{
		publicJamsLatest.mu.Lock()
		defer publicJamsLatest.mu.Unlock()
		latestJamData = publicJamsLatest.publicJamsLatestData
	}

	statusResponse := StatusResponse{
		1,
		true,
		time.Now().UnixMilli(),
		latestJamData.LastChangeTimestamp,
		humanize.Time(time.Unix(0, latestJamData.LastChangeTimestamp*int64(1000000))),
		latestJamData.LastChangeUser,
		latestJamData.LastChangeJam,
	}

	handlerEmitJson(httpResponse, statusResponse)
}

type CosmidManifestEntry struct {
	CID      string `json:"cosmid"`
	CDB      string `json:"couch"`
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

	SysLog.Info("Manifest requested", zap.String("RemoteAddr", r.RemoteAddr), zap.Int("Count", manifestResponse.Count))

	cosmidList := CurrentJamManifest.GetCOSMIDS()
	idBank := SysBankIDs.Bank()

	for _, cosmid := range cosmidList {
		cosmidName, ok := CurrentJamManifest.NameFromCOSMID(cosmid)
		if !ok {
			SysLog.Info("Manifest error", zap.String("COSMID", cosmid))
			break
		}
		cosmidPublic, ok := CurrentJamManifest.COSMIDJamIsPublic(cosmid)
		if !ok {
			SysLog.Info("Manifest public check error", zap.String("COSMID", cosmid))
			break
		}

		couchCID, ok := idBank.Entries[cosmid]
		if !ok {
			SysLog.Info("Manifest couch check error", zap.String("COSMID", cosmid))
			break
		}

		manifestResponse.Data = append(manifestResponse.Data, CosmidManifestEntry{cosmid, couchCID.CouchID, cosmidName, cosmidPublic})
	}

	handlerEmitJson(httpResponse, manifestResponse)
}
