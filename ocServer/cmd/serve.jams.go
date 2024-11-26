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
	"sync"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// -----------------------------------------------------------------------------------------------------------------------------------
// return from rifffsByCreateTime et al
type JamRiffRow struct {
	ID   string      `json:"id"`
	Data JamRiffData `json:"doc"`
}
type JamRiffView struct {
	TotalRows int          `json:"total_rows"`
	Offset    int          `json:"offset"`
	Rows      []JamRiffRow `json:"rows"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
type JamCuratedData struct {
	JamLongID  string `json:"jamId"` // "4ff9a55ab372aa8ced189925443d390501cbdb3e34744015a78dec0b3110ba09" ( band606301f6f2 )
	JamCouchID string
	JamName    string      `json:"name"`    // "5/4"
	Bio        string      `json:"bio"`     // "Keep the time signature the same. Tempo changes ok. think , Dave Brubeck Quartet – “Take Five”",
	ImageURL   string      `json:"image"`   // "https://endlesss.ams3.digitaloceanspaces.com/attachments/avatars/band606301f6f2"
	Owner      string      `json:"owner"`   //
	Members    []string    `json:"members"` //
	Riff       JamRiffData `json:"rifff"`
}
type JamCuratedResponse struct {
	Okay bool             `json:"ok"`
	Data []JamCuratedData `json:"data"`
}

type publicJamsLatestData struct {
	LastChangeTimestamp int64 // timestamp of the newest public riff; returned in server heartbeat
	LastChangeUser      string
	LastChangeJam       string
}
type PublicJamsLatest struct {
	publicJamsLatestData
	mu sync.Mutex
}

var publicJamsResponse *JamCuratedResponse // global instance of the public jams data, updated in a goroutine
var publicJamsLatest PublicJamsLatest

// -----------------------------------------------------------------------------------------------------------------------------------
// fetch the most recent single riff from the given jam
// user_appdata$xxxxx/_design/types/_view/rifffsByCreateTime?descending=true&limit=1&include_docs=true
func getRiffHeadDataFromJam(couchID string, couchClient *kivik.Client) (*JamRiffData, error) {

	jamDb := couchClient.DB(fmt.Sprintf("user_appdata$%s", couchID))
	resultSet := jamDb.Query(context.TODO(), "types", "rifffsByCreateTime", kivik.Params(map[string]interface{}{
		"descending":   true,
		"limit":        1,
		"include_docs": true,
	}))
	defer resultSet.Close()

	var resultData JamRiffData
	if !resultSet.Next() {
		return &resultData, nil
	}
	if err := resultSet.ScanDoc(&resultData); err != nil {
		return nil, err
	}
	if resultSet.Err() != nil {
		return nil, resultSet.Err()
	}

	return &resultData, nil
}

// -----------------------------------------------------------------------------------------------------------------------------------
func collectPublicJamStates(verboseOutput bool) {

	// ding dong
	couchClient, err := connectToCouchDB()
	if err != nil {
		SysLog.Error("[PublicJams] Connection to CouchDB failed", zap.Error(err))
		return
	}
	defer couchClient.Close()

	// fetch our current most-recent-timestamp for public riffs
	latestData := publicJamsLatestData{}
	latestData = publicJamsLatest.publicJamsLatestData

	for i := range publicJamsResponse.Data {

		curatedRiff := &publicJamsResponse.Data[i]

		// default to empty data, in case the following fails
		curatedRiff.Riff = JamRiffData{}

		// yank the most recent riff document from the jam via couch
		headRiff, err := getRiffHeadDataFromJam(curatedRiff.JamCouchID, couchClient)
		if err != nil {
			SysLog.Error("[PublicJams] getRiffHeadDataFromJam() failure", zap.String("CouchID", curatedRiff.JamCouchID), zap.Error(err))
		} else {
			curatedRiff.Riff = *headRiff

			// keep track of the most recent riff committed to a public jam
			if headRiff.Created > latestData.LastChangeTimestamp {
				latestData.LastChangeTimestamp = headRiff.Created
				latestData.LastChangeUser = headRiff.UserName
				latestData.LastChangeJam = curatedRiff.JamName
			}
		}

		if verboseOutput {
			SysLog.Info("Updated latest riff data", zap.String("CouchID", curatedRiff.JamCouchID), zap.String("From", curatedRiff.Riff.UserName), zap.Int64("Ts", curatedRiff.Riff.Created))
		}
	}

	// swap out the latest timestamp we found
	{
		publicJamsLatest.mu.Lock()
		defer publicJamsLatest.mu.Unlock()

		publicJamsLatest.publicJamsLatestData = latestData
	}
}

// -----------------------------------------------------------------------------------------------------------------------------------
// goroutine worker that pulls the latest riffs for the public jams every <time period> and re-encodes them into the
// static publicJamsResponse structure
var jamStateSema = semaphore.NewWeighted(int64(1))

func backgroundJamStateUpdater(chanStopWork <-chan struct{}) {

	SysLog.Info("backgroundJamStateUpdater launched")
	ctx := context.Background()
	for {
		// lock and update the public structure
		if err := jamStateSema.Acquire(ctx, 1); err != nil {
			SysLog.Error("background jam state thread error", zap.Error(err))
		} else {
			collectPublicJamStates(false)
			jamStateSema.Release(1)
		}

		if len(chanStopWork) != 0 {
			SysLog.Info("closing background jam state update worker")
			return
		}

		// TODO data drive the interval
		time.Sleep(30 * time.Second)
	}
}

// -----------------------------------------------------------------------------------------------------------------------------------
// return the current public jams
func HandlerJamCurated(httpResponse http.ResponseWriter, r *http.Request) {

	// grab a lock on the public jam data, ensuring we don't serialise a copy that is in the
	// process of being updated by backgroundJamStateUpdater() goroutine
	if err := jamStateSema.Acquire(context.TODO(), 1); err != nil {
		SysLog.Error("acquiring jam state sema failed", zap.Error(err))
		httpResponse.WriteHeader(http.StatusInternalServerError)
	} else {

		httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
		httpResponse.WriteHeader(http.StatusOK)
		json.NewEncoder(httpResponse).Encode(publicJamsResponse)

		jamStateSema.Release(1)
	}
}

// -----------------------------------------------------------------------------------------------------------------------------------
// returned format from user_appdata$xxxxxx/_design/membership/_view/getMembership
type MyJamMembership struct {
	ID          string `json:"_id"`           // "band30e45032ca"
	JoinDateISO string `json:"join_date_iso"` // "2021-05-07T19:48:58.279Z"
}

// -----------------------------------------------------------------------------------------------------------------------------------
// endpoint /jam/my-jams
// return the jam data that the user is subscribed to
func HandlerJamMyJams(httpResponse http.ResponseWriter, r *http.Request) {

	// grab user in use from headers
	authUsername, _, err := decodeAccountAuthBearer(r)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusForbidden)
		return
	}

	// open a line to couch
	couchClient, err := connectToCouchDB()
	if err != nil {
		SysLog.Error("[MyJams] Connection to CouchDB failed", zap.String("Username", authUsername), zap.Error(err))
		http.Error(httpResponse, err.Error(), http.StatusInternalServerError)
		return
	}
	defer couchClient.Close()

	var memberships []MyJamMembership

	// scrape the Band entries for a list of joined jams via the getMembership view
	userDb := couchClient.DB(fmt.Sprintf("user_appdata$%s", authUsername))
	resultSet := userDb.Query(context.TODO(), "membership", "getMembership", kivik.Params(map[string]interface{}{
		"include_docs": true,
	}))
	defer resultSet.Close()
	// scan them out into a local array of successful pulls, log but ignore any that error out
	for resultSet.Next() {
		var doc MyJamMembership
		if err := resultSet.ScanDoc(&doc); err != nil {
			SysLog.Error("[MyJams] ResultSet ScanDoc failure", zap.String("Username", authUsername), zap.Error(err))
		} else {
			memberships = append(memberships, doc)
		}
	}
	if resultSet.Err() != nil {
		SysLog.Error("[MyJams] ResultSet general failure", zap.String("Username", authUsername), zap.Error(resultSet.Err()))
	}

	var convertedMemberships []JamCuratedData

	// expand those singular couch IDs into more fully formed data to return back
	for _, v := range memberships {

		longID, idOK := SysBankIDs.LongFromCouch(v.ID)
		if !idOK {
			SysLog.Error("[MyJams] Failed to resolve long ID for couch ID", zap.String("CouchID", v.ID))
			continue
		}
		jamName, nameOK := CurrentJamManifest.NameFromCouch(v.ID)
		if !nameOK {
			SysLog.Error("[MyJams] Unknown CouchID passed to NameFromCouch()", zap.String("CouchID", v.ID))
			continue
		}

		// go fetch the most recent riff data, used to sort the tiles in the Studio UI
		headRiff, err := getRiffHeadDataFromJam(v.ID, couchClient)
		if err != nil {
			SysLog.Error("[MyJams] getRiffHeadDataFromJam() failure", zap.String("Username", authUsername), zap.Error(err))
		}

		// plug it all together
		entry := JamCuratedData{}
		entry.JamLongID = longID
		entry.JamCouchID = v.ID
		entry.Bio = "" // dont think this is used in the UI at this point so not bothering
		entry.JamName = jamName
		entry.ImageURL = fmt.Sprintf("%s/api/v3/image/avatars/%s", getCosmServerExternalHost(), v.ID)
		entry.Members = []string{authUsername}

		// might have failed, that's fine; worst case it will just show a blank spot where the riff would be
		if headRiff != nil {
			entry.Riff = *headRiff
		}

		convertedMemberships = append(convertedMemberships, entry)
	}

	userJams := &JamCuratedResponse{
		Okay: true,
		Data: convertedMemberships,
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(userJams)
}
