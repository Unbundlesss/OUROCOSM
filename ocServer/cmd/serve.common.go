//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/go-kivik/kivik/v4"
	"github.com/homedepot/flop"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	ContentTypeApplicationJson string = "application/json"
	HeaderNameContentType      string = "Content-Type"
)
const (
	CouchKnownDatabase_AppClientConfig string = "app_client_config"
	CouchKnownDocument_BandsJoinable   string = "bands:joinable"
)

func handlerEmitJson(httpResponse http.ResponseWriter, objectToSend any) {
	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(objectToSend)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// structure of the jams.json file that is loaded on boot to configure the server's jam knowledge
type CosmServerJamDecl struct {
	COSMID  string   `json:"cosmid"`
	Name    string   `json:"name"`
	Bio     string   `json:"bio"`
	Members []string `json:"members"`
}
type CosmServerJamData struct {
	Public  []CosmServerJamDecl `json:"public"`
	Private []CosmServerJamDecl `json:"private"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
// document format for the Profile record, a single document of id "Profile" that is used to identify a jam database to Studio;
// these are kept in sync with data from jams.json each time the server boots
type JamDatabaseProfileData struct {
	Bio         string `json:"bio"`
	Created     int64  `json:"created"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type"`
}
type JamDatabaseProfileUpdate struct {
	Rev string `json:"_rev"`
	JamDatabaseProfileData
}

// -----------------------------------------------------------------------------------------------------------------------------------
// document written into user databases to denote that users' My Jams membership to the jam in question (denoted by doc id)
type JamMembershipRecord struct {
	JoinDate    int64    `json:"join_date"`     // 1620416938300
	JoinDateISO string   `json:"join_date_iso"` // "2021-05-07T19:48:58.279Z"
	Lists       []string `json:"lists"`
	Type        string   `json:"type"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
// document found in app_client_config describing available jams using couch IDs; primarily used in the `bands:joinable` document
// to publish public jams that can be joined - and if this is not kept up to date, Endlesss doesn't seem to let people join public jams properly
type AppClientConfigBands struct {
	BandIDs            []string `json:"band_ids"`
	Joinable           bool     `json:"joinable"`
	BannerImage        string   `json:"bannerImage"`
	DesktopBannerImage string   `json:"desktopBannerImage"`
}
type AppClientConfigBandsUpdate struct {
	Rev string `json:"_rev"`
	AppClientConfigBands
}

// -----------------------------------------------------------------------------------------------------------------------------------
// lookups built from the jam manifest file, mapping ids to chosen public names
type JamManifest struct {
	couchToName    map[string]string
	cosmidToName   map[string]string
	cosmidIsPublic map[string]bool
}

func (jman JamManifest) NameFromCouch(couchID string) (string, bool) {
	result, ok := jman.couchToName[couchID]
	return result, ok
}
func (jman JamManifest) NameFromCOSMID(cosmid string) (string, bool) {
	result, ok := jman.cosmidToName[cosmid]
	return result, ok
}
func (jman JamManifest) COSMIDJamIsPublic(cosmid string) (bool, bool) {
	result, ok := jman.cosmidIsPublic[cosmid]
	return result, ok
}
func (jman JamManifest) NumberOfCOSMIDs() int {
	return len(jman.cosmidToName)
}
func (jman JamManifest) GetCOSMIDS() []string {
	keys := make([]string, 0, len(jman.cosmidToName))
	for k := range jman.cosmidToName {
		keys = append(keys, k)
	}
	return keys
}

// our current stack of known jams
var CurrentJamManifest *JamManifest

// -----------------------------------------------------------------------------------------------------------------------------------
func getCosmServerExternalHost() string {

	return fmt.Sprintf("%s://%s:%s",
		viper.GetString(cConfigCosmScheme),
		viper.GetString(cConfigCosmExternalHost),
		viper.GetString(cConfigCosmExternalPort),
	)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// run each time the server boots - process data from the jams.json manifest; do boot-time checks, refresh profiles, etc
func performJamPreflight(couchClient *kivik.Client, jamDecl CosmServerJamDecl, isPublic bool, jamManifest *JamManifest) *JamCuratedData {

	idBank := SysBankIDs.Bank()

	lutID, ok := idBank.Entries[jamDecl.COSMID]
	if !ok {
		SysLog.Fatal("Unable to resolve COSMID to Endlesss jam IDs", zap.String("COSMID", jamDecl.COSMID))
	}

	SysLog.Info("Registering Jam",
		zap.Bool("IsPublic", isPublic),
		zap.String("COSMID", jamDecl.COSMID),
		zap.String("Name", jamDecl.Name),
		zap.String("CouchID", lutID.CouchID),
	)

	// check to see if the jam database exists yet - if not, ask for a new one
	jamExists, err := doesJamDatabaseExist(couchClient, lutID.CouchID)
	if err != nil {
		SysLog.Fatal("Fatal error checking jam database state", zap.String("COSMID", jamDecl.COSMID), zap.Error(err))
	}
	if !jamExists {
		err = createDefaultPublicJamDatabase(couchClient, lutID.CouchID)
		if err != nil {
			SysLog.Fatal("Fatal error creating new jam", zap.String("COSMID", jamDecl.COSMID), zap.Error(err))
		}
	}

	// fun fact the ImageURL is totally ignored by Endlesss, it seems. we need to clone the
	// COSMID jpegs into `band##########` files in the avatars root so they get found properly
	//
	// from <root>/avatars_source/<cosmid> -> <root>/avatars/<band_id>
	//
	avatarImageFromPath := path.Join(path.Join(cmdServeRootPath, "avatars_source"), jamDecl.COSMID+".jpg")
	avatarImageToPath := path.Join(path.Join(cmdServeRootPath, "avatars"), lutID.CouchID)
	err = flop.SimpleCopy(avatarImageFromPath, avatarImageToPath)
	if err != nil {
		SysLog.Fatal("Jam avatar copy error", zap.String("COSMID", jamDecl.COSMID), zap.Error(err))
	}

	// snag the file stat for the jam avatar, we'll use that for the creation time
	avatarInfo, err := os.Stat(avatarImageFromPath)
	if err != nil {
		SysLog.Fatal("Jam avatar stat() error", zap.String("COSMID", jamDecl.COSMID), zap.Error(err))
	}

	// check on the database entry for this jam - and update the Profile document automatically each time with
	// any name/bio changes .. this also checks that the database exists etc
	jamDb := couchClient.DB(fmt.Sprintf("user_appdata$%s", lutID.CouchID))
	if jamDb.Err() != nil {
		SysLog.Fatal("Error checking jam database", zap.String("COSMID", jamDecl.COSMID), zap.String("CouchID", lutID.CouchID), zap.Error(jamDb.Err()))
	}
	// pull the current Profile doc
	var currentJamProfile JamDatabaseProfileUpdate
	err = jamDb.Get(context.TODO(), "Profile").ScanDoc(&currentJamProfile)
	if err != nil {
		SysLog.Fatal("Unable to fetch jam Profile document", zap.String("COSMID", jamDecl.COSMID), zap.String("CouchID", lutID.CouchID), zap.Error(err))
	}
	// if we have new data to write in, go update that document
	if currentJamProfile.DisplayName != jamDecl.Name || currentJamProfile.Bio != jamDecl.Bio {

		// repopulate to the latest data
		currentJamProfile.Created = avatarInfo.ModTime().UnixMilli()
		currentJamProfile.DisplayName = jamDecl.Name
		currentJamProfile.Bio = jamDecl.Bio
		currentJamProfile.Type = "Profile"

		// note that we're changing stuff
		SysLog.Info("Updating jam Profile document ...", zap.String("COSMID", jamDecl.COSMID))

		_, err = jamDb.Put(context.TODO(), "Profile", currentJamProfile)
		if err != nil {
			SysLog.Fatal("Unable to update jam Profile document", zap.String("COSMID", jamDecl.COSMID), zap.Error(err))
		}
	}

	// for private jams, update member records to add them to the users' My Jams lists
	if !isPublic {
		for _, v := range jamDecl.Members {
			userDb := couchClient.DB(fmt.Sprintf("user_appdata$%s", v))

			existingMembership := &JamMembershipRecord{}
			err = userDb.Get(context.TODO(), lutID.CouchID).ScanDoc(existingMembership)

			// no error - means the document already exists, nothing for us to do
			if err == nil {
				continue
			}

			// kind of stupid, we have to check on strings to discover what *kind* of missing doc we fail to find?
			if strings.Contains(err.Error(), "Not Found: missing") {
				// user exists, membership doesn't
				SysLog.Info("Adding membership document", zap.String("COSMID", jamDecl.COSMID), zap.String("Username", v))

				newMembership := JamMembershipRecord{
					JoinDate:    time.Now().UnixMilli(),
					JoinDateISO: time.Now().Format(time.RFC3339),
					Lists:       []string{"myJams"},
					Type:        "Band",
				}
				_, err = userDb.Put(context.TODO(), lutID.CouchID, newMembership)
				if err != nil {
					SysLog.Error("Unable to insert membership document", zap.String("COSMID", jamDecl.COSMID), zap.String("Username", v), zap.Error(err))
				}

			} else if strings.Contains(err.Error(), "Not Found: Database does not exist") {
				// user doesn't exist
				SysLog.Error("User does not exist", zap.String("COSMID", jamDecl.COSMID), zap.String("Username", v))
			}
		}
	}

	// final data block in a format for Studio, if this jam is being returned to the user
	entry := JamCuratedData{}
	entry.JamLongID = lutID.LongID
	entry.JamCouchID = lutID.CouchID
	entry.Bio = jamDecl.Bio
	entry.JamName = jamDecl.Name
	entry.ImageURL = fmt.Sprintf("%s/api/v3/image/avatars/%s", getCosmServerExternalHost(), lutID.CouchID)
	entry.Members = jamDecl.Members

	jamManifest.couchToName[lutID.CouchID] = jamDecl.Name
	jamManifest.cosmidToName[jamDecl.COSMID] = jamDecl.Name
	jamManifest.cosmidIsPublic[jamDecl.COSMID] = isPublic

	return &entry
}

// -----------------------------------------------------------------------------------------------------------------------------------
// take in the jams.json manifest file and process it into internal datasets
func constructJamManifestFromData(jamData CosmServerJamData) *JamManifest {

	manifestResult := JamManifest{}
	manifestResult.couchToName = make(map[string]string)
	manifestResult.cosmidToName = make(map[string]string)
	manifestResult.cosmidIsPublic = make(map[string]bool)

	publicJamsResponse = new(JamCuratedResponse)
	publicJamsResponse.Okay = true

	// ring up couch, we will do some validation of databases while we load this gunk
	couchClient, err := connectToCouchDB()
	if err != nil {
		SysLog.Fatal("Connection to CouchDB failed", zap.Error(err))
	}
	defer couchClient.Close()

	// keep a list of public jam IDs to write into ACC later
	var joinablePublicBandIds []string

	SysLog.Info("Preflight - Public")
	for _, v := range jamData.Public {
		entry := performJamPreflight(couchClient, v, true, &manifestResult)
		publicJamsResponse.Data = append(publicJamsResponse.Data, *entry)

		joinablePublicBandIds = append(joinablePublicBandIds, entry.JamCouchID)
	}
	SysLog.Info("Preflight - Private")
	for _, v := range jamData.Private {
		performJamPreflight(couchClient, v, false, &manifestResult)
	}

	// sort the list of public IDs to try and keep them stable across runs
	sort.Strings(joinablePublicBandIds)

	// grab the ACC, automatically update the joinable bands list if we need to - this needs to be kept in sync with
	// the public IDs returned by /jam/curated otherwise Endlesss will display the publics but not allow you to actually enter one
	accExists, err := doesDatabaseExist(couchClient, CouchKnownDatabase_AppClientConfig)
	if err != nil {
		SysLog.Fatal("Failed to examine app client config database", zap.Error(err))
	}
	if accExists {

		accDb := couchClient.DB(CouchKnownDatabase_AppClientConfig)

		// read out the current joinable document to see if we need to update it
		var currentBandsJoinable AppClientConfigBandsUpdate
		err = accDb.Get(context.TODO(), CouchKnownDocument_BandsJoinable).ScanDoc(&currentBandsJoinable)
		if err != nil {
			SysLog.Fatal("Unable to fetch bands:joinable document for update", zap.Error(err))
		}

		// embed current state
		currentBandsJoinable.Joinable = true
		currentBandsJoinable.BandIDs = joinablePublicBandIds
		currentBandsJoinable.BannerImage = fmt.Sprintf("%s/static/cosm_banner_mobile.jpg", getCosmServerExternalHost())
		currentBandsJoinable.DesktopBannerImage = fmt.Sprintf("%s/static/cosm_banner_desktop.jpg", getCosmServerExternalHost())

		_, err = accDb.Put(context.TODO(), CouchKnownDocument_BandsJoinable, currentBandsJoinable)
		if err != nil {
			SysLog.Fatal("Unable to update bands:joinable document", zap.Error(err))
		}

	} else {
		// eventually - fully build and populate the ACC
		SysLog.Fatal("App client config database does not exist, Couch needs configuring")
	}

	return &manifestResult
}
