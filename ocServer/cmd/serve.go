//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/sollniss/graceful"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"go.uber.org/zap"
)

var cmdServeRootPath = ""

// per ourocosm.server.yaml
const cConfigCosmScheme string = "cosm.scheme"
const cConfigCosmInternalHost string = "cosm.internal-host"
const cConfigCosmInternalPort string = "cosm.internal-port"
const cConfigCosmExternalHost string = "cosm.external-host"
const cConfigCosmExternalPort string = "cosm.external-port"
const cConfigCosmFourCC string = "cosm.fourcc"
const cConfigCosmAPIPrefix string = "cosm.api-prefix"
const cConfigCosmAPIAuth string = "cosm.api-auth"

// -----------------------------------------------------------------------------------------------------------------------------------
func HandlerDefault(httpResponse http.ResponseWriter, r *http.Request) {
	SysLog.Warn("Unhandled request", zap.String("Method", r.Method), zap.String("URI", r.RequestURI), zap.String("RemoteAddr", r.RemoteAddr))
	httpResponse.WriteHeader(http.StatusNotFound)
}

// -----------------------------------------------------------------------------------------------------------------------------------
var SecuredApiCredentials map[string]string

// simple security gateway around chosen API endpoints, uses user/pass as given by config api-auth table
// intention being that we might have trusted tools working with some of those endpoints, keep them tucked away from scraping/casual abuse
func SecuredApiAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()

		loginAccepted := false
		if ok {
			expectedPass, ok := SecuredApiCredentials[user]
			if ok {
				loginAccepted = subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) == 1
			}
		}

		if !loginAccepted {
			w.Header().Set("WWW-Authenticate", `Basic realm="ourocosm-api"`)
			http.Error(w, "Unavailable", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// -----------------------------------------------------------------------------------------------------------------------------------
func runCosmServer() {

	// launch the background jam state worker routine
	bgJamWorker := make(chan struct{}, 1)
	go backgroundJamStateUpdater(bgJamWorker)
	defer close(bgJamWorker)

	router := mux.NewRouter()

	// some api functions are tucked away behind a server-side prefix with basic authentication
	apiPrefix := viper.GetString(cConfigCosmAPIPrefix)
	SecuredApiCredentials = viper.GetStringMapString(cConfigCosmAPIAuth)
	for k := range SecuredApiCredentials {
		SysLog.Info("API Access granted", zap.String("Username", k)) // list out API users just to keep an eye on them
	}

	// authentication
	router.HandleFunc("/auth/login", HandlerAuthLogin).Methods("POST")
	router.HandleFunc("/auth/session", HandlerAuthSession).Methods("GET")
	router.HandleFunc("/accounts/auth/remote-login/{authkey}", HandlerAuthRemote).Methods("GET") // stubbed out, remote/QR login not supported

	// notification
	router.HandleFunc("/notifications", HandlerNotifications).Methods("GET")
	router.HandleFunc("/api/notify/{username}", HandlerNotifyData).Methods("POST")

	// accounts
	router.HandleFunc("/accounts/profile", HandlerAccountsProfileGet).Methods("GET")
	router.HandleFunc("/accounts/profile", HandlerAccountsProfilePost).Methods("POST")
	router.HandleFunc("/accounts/{username}/profile", HandlerAccountsProfileSpecific).Methods("GET")
	router.HandleFunc("/accounts/{username}/following", HandlerAccountsFollowing).Methods("GET")

	// jams
	router.HandleFunc("/jam/curated", HandlerJamCurated).Methods("GET")
	router.HandleFunc("/jam/my-jams", HandlerJamMyJams).Methods("GET")
	router.HandleFunc("/api/band/{couchid}/listenlink", HandlerListenLink).Methods("GET")
	router.HandleFunc("/api/band/{couchid}/permalink", HandlerListenLink).Methods("GET") // re-use ListenLink, we have no other long-id we can supply
	router.HandleFunc("/api/band/create", HandlerJamCreate).Methods("POST")
	router.HandleFunc("/api/band/{couchid}/leave", HandlerJamLeave).Methods("POST")
	router.HandleFunc("/api/band/{longid}/listen", HandlerJamListenLong).Methods("POST")

	router.HandleFunc("/marketplace/collectible-jams/{longid}", HandlerMarketplace).Methods("GET") // null stub for this, arrives every time someone looks at a jam

	// sound packs
	router.HandleFunc("/sound-packs", HandlerSoundPacksGet).Methods("GET")
	router.HandleFunc("/sound-packs", HandlerSoundPacksPost).Methods("POST")
	router.HandleFunc("/sound-packs/presets", HandlerSoundPacksPresetsPost).Methods("POST")

	// custom bits
	router.HandleFunc("/cosm/v1/status", HandlerCosmStatus).Methods("GET") // return a basic heartbeat response indicating server is alive, server time, etc

	// custom bits that we want locked behind some kind of path obfuscation + user/pass visibility
	securedApi := router.PathPrefix(fmt.Sprintf("/cosm/v1/%s", apiPrefix)).Subrouter()
	securedApi.HandleFunc("/manifest", HandlerCosmManifest).Methods("GET") // return base details about COSMIDs in use
	securedApi.Use(SecuredApiAuth)

	// static data handling for avatars or generic images
	rootAvatarPath := path.Join(cmdServeRootPath, "avatars")
	router.PathPrefix("/api/v3/image/avatars/").Handler(http.StripPrefix("/api/v3/image/avatars/", http.FileServer(http.Dir(rootAvatarPath))))
	rootStaticPath := path.Join(cmdServeRootPath, "static")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(rootStaticPath))))

	// generic fallback to track unhandled requests
	router.PathPrefix("/").HandlerFunc(HandlerDefault)

	n := negroni.New()
	n.Use(negroni.NewRecovery())
	// n.Use(negroni.NewLogger())
	n.Use(NewServerIdent())
	n.Use(gzip.Gzip(gzip.BestSpeed))
	n.UseHandler(router)

	cosmAddressInternal := fmt.Sprintf("%s:%s", viper.GetString(cConfigCosmInternalHost), viper.GetString(cConfigCosmInternalPort))

	var httpServer = &http.Server{
		Handler:      n,
		WriteTimeout: time.Second * 5,
		ReadTimeout:  time.Second * 5,
		IdleTimeout:  time.Second * 10,
		Addr:         cosmAddressInternal,
	}

	SysLog.Info(fmt.Sprintf("Launching API server on %s", cosmAddressInternal))
	SysLog.Info(fmt.Sprintf("External API address is %s", getCosmServerExternalHost()))
	ctx := graceful.NotifyShutdown()
	err := graceful.ListenAndServe(ctx, httpServer, 60*time.Second)
	if err != nil {
		SysLog.Error("error during shutdown", zap.Error(err))
		return
	}

	SysLog.Info("Server graceful shutdown, toodles!")
}

// -----------------------------------------------------------------------------------------------------------------------------------
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the COSM API server",
	Long:  `Run the COSM API server`,
	Run: func(cmd *cobra.Command, args []string) {

		// check the chosen root path exists
		_, err := os.Stat(cmdServeRootPath)
		if os.IsNotExist(err) {
			SysLog.Fatal("Server root path not found", zap.String("Path", cmdServeRootPath))
		}
		SysLog.Info("Server root path set", zap.String("Path", cmdServeRootPath))

		// check we have all the configuration entries we need to formulate a connection string
		configChecks := []string{
			cConfigCosmScheme,
			cConfigCosmExternalHost,
			cConfigCosmExternalPort,
			cConfigCosmInternalHost,
			cConfigCosmInternalPort,
			cConfigCosmFourCC,
			cConfigCosmAPIPrefix,
		}
		for _, v := range configChecks {
			if !viper.IsSet(v) || len(viper.GetString(v)) == 0 {
				SysLog.Fatal(fmt.Sprintf("configuration key '%s' missing or empty", v))
			}
		}

		// check the identity string
		fourccCheck := viper.GetString(cConfigCosmFourCC)
		if !isASCII(fourccCheck) {
			SysLog.Fatal("Server FOURCC identity contains invalid characters", zap.String("fourcc", fourccCheck))
		}
		if len(fourccCheck) != 4 {
			SysLog.Fatal("Server FOURCC identity should be 4 characters long", zap.String("fourcc", fourccCheck))
		}

		// fetch the jam mapping data from disk (also therefore checks the chosen root path out)
		manifestPath := path.Join(cmdServeRootPath, "jams.json")
		manifestJsonData, err := os.ReadFile(manifestPath)
		if err != nil {
			SysLog.Fatal("Unable to load jam manifest JSON", zap.Error(err), zap.String("Path", manifestPath))
		}
		var jamData CosmServerJamData
		err = json.Unmarshal(manifestJsonData, &jamData)
		if err != nil {
			SysLog.Fatal("Unable to parse jam manifest JSON", zap.Error(err), zap.String("Path", manifestPath))
		}

		// utilise that loaded jam manifest
		CurrentJamManifest = constructJamManifestFromData(jamData)

		// populate the jam manifest cache with the latest riff data to begin with; this can then be updated
		// in the background every so often
		collectPublicJamStates(true)

		// lets go!
		runCosmServer()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&cmdServeRootPath, "root", "r", "", "(required) root file path for server assets")
	serveCmd.MarkFlagRequired("root")
}
