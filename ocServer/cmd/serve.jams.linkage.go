//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type ListenLinkData struct {
	Url     string `json:"url"`
	Host    string `json:"host"`
	Path    string `json:"path"`
	CouchID string `json:"band_id"`
	JamName string `json:"band_name"`
}
type ListenLinkResponse struct {
	Result   string         `json:"result"`
	Data     ListenLinkData `json:"data"`
	Messages []string       `json:"messages"`
	Errors   []string       `json:"errors"`
	Links    []string       `json:"links"`
	Version  int            `json:"version"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
// responses for both listenlink and permalink requests, looking for the long id from a given couch id
func HandlerListenLink(httpResponse http.ResponseWriter, r *http.Request) {

	// get the couch id given in the request url
	vars := mux.Vars(r)
	var sourceCouchId = vars["couchid"]
	if len(sourceCouchId) < 4 {
		http.Error(httpResponse, "Unknown CouchID for listenlink request", http.StatusBadRequest)
		return
	}

	// look up the long ID
	sourceLongId, idOK := SysBankIDs.LongFromCouch(sourceCouchId)
	if !idOK {
		SysLog.Error("Unknown CouchID passed to listenlink", zap.String("CouchID", sourceCouchId))
		http.Error(httpResponse, "CouchID unrecognised", http.StatusBadRequest)
		return
	}

	jamName, nameOK := CurrentJamManifest.NameFromCouch(sourceCouchId)
	if !nameOK {
		SysLog.Error("Unknown CouchID passed to NameFromCouch()", zap.String("CouchID", sourceCouchId))
		http.Error(httpResponse, "Jam name not found", http.StatusBadRequest)
		return
	}

	listenLinkResponse := &ListenLinkResponse{
		Result: "success",
		Data: ListenLinkData{
			Url:     fmt.Sprintf("%s/jam/%s/listen", getCosmServerExternalHost(), sourceLongId),
			Host:    getCosmServerExternalHost(),
			Path:    fmt.Sprintf("/jam/%s/listen", sourceLongId),
			CouchID: sourceCouchId,
			JamName: jamName,
		},
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(listenLinkResponse)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// this is triggered by trying to join a jam ... ?
func HandlerJamListenLong(httpResponse http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	SysLog.Info("Listen link requested", zap.String("LongID", vars["longid"]))

	defaultOkayResponse := "{\"ok\":true,\"message\":\"null\"}"

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	httpResponse.Write([]byte(defaultOkayResponse))
}

// -----------------------------------------------------------------------------------------------------------------------------------
// no support for the marketplace, thanks
func HandlerMarketplace(httpResponse http.ResponseWriter, r *http.Request) {

	defaultMissingMarketplaceResponse := "{\"ok\":false,\"code\":404,\"type\":null,\"error\":404,\"message\":\"null\"}"

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	httpResponse.Write([]byte(defaultMissingMarketplaceResponse))
}
