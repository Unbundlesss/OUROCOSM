//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type AccountsFollowingData struct {
	FollowingCount int      `json:"followingCount"`
	FollowersCount int      `json:"followersCount"`
	Following      []string `json:"following"`
	Followers      []string `json:"followers"`
}
type AccountsFollowingResponse struct {
	Data AccountsFollowingData `json:"data"`
}

func HandlerAccountsFollowing(httpResponse http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	SysLog.Info("Account Following", zap.String("User", vars["username"]), zap.String("URI", r.RequestURI))

	followingResponse := &AccountsFollowingResponse{
		Data: AccountsFollowingData{0, 0, []string{}, []string{}},
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(followingResponse)
}
