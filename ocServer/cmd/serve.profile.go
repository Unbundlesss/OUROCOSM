//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type AccountsProfileData struct {
	EmailAddress string `json:"email"`
	Biography    string `json:"bio"`
	AvatarUrl    string `json:"avatarUrl"`
}
type AccountsProfileResponse struct {
	Data AccountsProfileData `json:"data"`
}

// sent as the body with POSTing to /accounts/profile
type AccountsProfileModify struct {
	Account       string `json:"account"`
	DisplayName   string `json:"displayName"`
	ExternalLinks struct {
		Discord    string `json:"discord"`
		Instagram  string `json:"instagram"`
		Tiktok     string `json:"tiktok"`
		Youtube    string `json:"youtube"`
		Soundcloud string `json:"soundcloud"`
		Bandlab    string `json:"bandlab"`
		Twitter    string `json:"twitter"`
		Facebook   string `json:"facebook"`
		Twitch     string `json:"twitch"`
		Spotify    string `json:"spotify"`
		Bandcamp   string `json:"bandcamp"`
		Website    string `json:"website"`
	} `json:"externalLinks"`
	Email string `json:"email"`
	Bio   string `json:"bio"`
}
type AccountsProfileModifyResponse struct {
	Okay bool                  `json:"ok"`
	Data AccountsProfileModify `json:"data"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
func genericProfileResponse(profileName string, httpResponse http.ResponseWriter) {

	couchClient, err := connectToCouchDB()
	if err != nil {
		SysLog.Error("Connection to CouchDB failed", zap.Error(err))
		http.Error(httpResponse, "Database connection failure", http.StatusInternalServerError)
		return
	}
	defer couchClient.Close()

	userExtras, err := fetchUserExtrasFromCouch(couchClient, profileName)
	if err != nil {
		SysLog.Error("Unable to fetch userdata", zap.Error(err), zap.String("User", profileName))
		http.Error(httpResponse, "Database read failure", http.StatusInternalServerError)
		return
	}

	profileResponse := &AccountsProfileResponse{
		Data: AccountsProfileData{
			"null@void.com",
			userExtras.Bio,
			fmt.Sprintf("%s/api/v3/image/avatars/%s", getCosmServerExternalHost(), profileName),
		},
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(profileResponse)
}

// -----------------------------------------------------------------------------------------------------------------------------------
func HandlerAccountsProfileGet(httpResponse http.ResponseWriter, r *http.Request) {

	authUsername, _, err := decodeAccountAuthBearer(r)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusForbidden)
		return
	}

	SysLog.Info("Loading account profile", zap.String("User", authUsername))
	genericProfileResponse(authUsername, httpResponse)

}

// -----------------------------------------------------------------------------------------------------------------------------------
func HandlerAccountsProfilePost(httpResponse http.ResponseWriter, r *http.Request) {

	authUsername, _, err := decodeAccountAuthBearer(r)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusForbidden)
		return
	}

	var newAccountData AccountsProfileModify

	// fetch and decode the body containing the new account data
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		SysLog.Error("Failed to ready body text in account update", zap.Error(err), zap.String("User", authUsername))
		http.Error(httpResponse, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(buf, &newAccountData)
	if err != nil {
		SysLog.Error("Failed to decode data for account update", zap.Error(err), zap.String("User", authUsername))
		http.Error(httpResponse, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO
	// write data to db

	SysLog.Info("Updating account profile", zap.String("User", authUsername))

	// Studio will crash unless we get this response right!
	// respond with the same data we got sent, wrapped in the standard "okay" header
	response := AccountsProfileModifyResponse{
		true,
		newAccountData,
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(response)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// response for when looking at other people's profiles, by username
func HandlerAccountsProfileSpecific(httpResponse http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	profileToLoad := vars["username"]
	if len(profileToLoad) == 0 {
		http.Error(httpResponse, "Invalid username for profile", http.StatusInternalServerError)
		return
	}

	SysLog.Info("Loading specific account profile", zap.String("User", profileToLoad))
	genericProfileResponse(profileToLoad, httpResponse)
}
