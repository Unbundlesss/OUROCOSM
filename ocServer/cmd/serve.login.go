//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type AuthLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthLoginUserDB struct {
	AppData string `json:"appdata"`
}
type AuthLoginProfile struct {
	DisplayName string `json:"displayName"`
}
type AuthLoginResponse struct {
	Issued    int64            `json:"issued"`
	Expires   int64            `json:"expires"`
	Provider  string           `json:"provider"`
	IPAddress string           `json:"ip"`
	Token     string           `json:"token"`
	Password  string           `json:"password"`
	UserID    string           `json:"user_id"`
	Roles     []string         `json:"roles"`
	UserDBs   AuthLoginUserDB  `json:"userDBs"`
	Profile   AuthLoginProfile `json:"profile"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
func HandlerAuthLogin(httpResponse http.ResponseWriter, r *http.Request) {

	var authLoginRequest AuthLoginRequest

	err := json.NewDecoder(r.Body).Decode(&authLoginRequest)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusBadRequest)
		return
	}
	SysLog.Info("Login flow", zap.String("User", authLoginRequest.Username))

	couchClient, err := connectToCouchDB()
	if err != nil {
		SysLog.Error("Connection to CouchDB failed", zap.Error(err))
		http.Error(httpResponse, "Database connection failure", http.StatusInternalServerError)
		return
	}
	defer couchClient.Close()

	userExtras, err := fetchUserExtrasFromCouch(couchClient, authLoginRequest.Username)
	if err != nil {
		SysLog.Error("Unable to fetch userdata", zap.Error(err), zap.String("User", authLoginRequest.Username))
		http.Error(httpResponse, "Database read failure", http.StatusInternalServerError)
		return
	}

	if userExtras.Login != authLoginRequest.Password {
		SysLog.Error("Invalid password", zap.String("User", authLoginRequest.Username))
		http.Error(httpResponse, "Invalid password", http.StatusUnauthorized)
		return
	}

	userToken := generateInternalCouchUserPassword(authLoginRequest.Username)

	resolvedCouchExternalIP := viper.GetString(cConfigCouchExternalHost)
	ips, err := net.LookupIP(resolvedCouchExternalIP)
	if err == nil {
		for _, ip := range ips {
			if ip.To4() != nil {
				resolvedCouchExternalIP = ip.String()
				break
			}
		}
	}

	userHomeDb := fmt.Sprintf("%s://%s:%s@%s:%s/user_appdata$%s",
		viper.GetString(cConfigCouchScheme),
		authLoginRequest.Username,
		userToken,
		resolvedCouchExternalIP,
		viper.GetString(cConfigCouchExternalPort),
		authLoginRequest.Username)

	SysLog.Info("Account Login",
		zap.String("Home", userHomeDb),
	)

	authLoginResponse := &AuthLoginResponse{
		Issued:    time.Now().UnixMilli(),
		Expires:   time.Now().UnixMilli() + 15000000000,
		Provider:  "local",
		IPAddress: r.RemoteAddr,
		Token:     authLoginRequest.Username,
		Password:  userToken,
		UserID:    authLoginRequest.Username,
		Roles:     []string{"user", "jammers", "admin"},
		UserDBs:   AuthLoginUserDB{userHomeDb},
		Profile:   AuthLoginProfile{authLoginRequest.Username},
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(authLoginResponse)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// no idea how to handle the QR login stuff, never used it; presumably we would end up returning similar to the above
func HandlerAuthRemote(httpResponse http.ResponseWriter, r *http.Request) {
	httpResponse.WriteHeader(http.StatusGone)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// heartbeat tick from Studio to check on our authentication status; smaller subsection of the full auth response

type AuthSessionResponse struct {
	Issued   int64    `json:"issued"`
	Expires  int64    `json:"expires"`
	Provider string   `json:"provider"`
	UserID   string   `json:"user_id"`
	Roles    []string `json:"roles"`
}

func HandlerAuthSession(httpResponse http.ResponseWriter, r *http.Request) {

	authUsername, _, err := decodeAccountAuthBearer(r)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusForbidden)
		return
	}

	authSessionResponse := &AuthSessionResponse{
		Issued:   time.Now().UnixMilli(),
		Expires:  time.Now().UnixMilli() + 15000000000, // the party never stops!
		Provider: "local",
		UserID:   authUsername,
		Roles:    []string{"user", "jammers", "admin"},
	}

	// note in server log that we have people around
	SysLog.Info("User heartbeat", zap.String("User", authUsername), zap.String("RemoteAddr", r.RemoteAddr))

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(authSessionResponse)
}
