//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	"github.com/go-kivik/kivik/v4/couchdb"
	"github.com/hymkor/go-lazy"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// per ourocosm.server.yaml
const cConfigCouchScheme string = "couchDB.scheme"
const cConfigCouchInternalHost string = "couchDB.internal-host"
const cConfigCouchInternalPort string = "couchDB.internal-port"
const cConfigCouchExternalHost string = "couchDB.external-host"
const cConfigCouchExternalPort string = "couchDB.external-port"
const cConfigCouchUser string = "couchDB.user"
const cConfigCouchPwd string = "couchDB.pwd"
const cConfigCouchSalt string = "couchDB.salt"

var CouchConnectionURI = lazy.Of[string]{
	New: func() string {

		SysLog.Info("Checking CouchDB configuration ...")

		// check we have all the configuration entries we need to formulate a connection string
		configChecks := []string{
			cConfigCouchScheme,
			cConfigCouchInternalHost,
			cConfigCouchInternalPort,
			cConfigCouchExternalHost,
			cConfigCouchExternalPort,
			cConfigCouchUser,
			cConfigCouchPwd,
		}
		for _, v := range configChecks {
			if !viper.IsSet(v) || len(viper.GetString(v)) == 0 {
				SysLog.Fatal("Configuration key missing or empty", zap.String("Key", v))
			}
		}

		couchURI := fmt.Sprintf("%s://%s:%s",
			viper.GetString(cConfigCouchScheme),
			viper.GetString(cConfigCouchInternalHost),
			viper.GetString(cConfigCouchInternalPort),
		)

		SysLog.Info("CouchDB configuration set", zap.String("URI", couchURI))

		return couchURI
	},
}

// -----------------------------------------------------------------------------------------------------------------------------------
// take care of connecting to couch via the viper config state, or die trying
func connectToCouchDB() (*kivik.Client, error) {
	return kivik.New("couch", CouchConnectionURI.Value(), couchdb.BasicAuth(viper.GetString(cConfigCouchUser), viper.GetString(cConfigCouchPwd)))
}

// -----------------------------------------------------------------------------------------------------------------------------------
// create a new jam database container with all its default design docs
func createNewJamDatabase(client *kivik.Client, jamName string) (*kivik.DB, error) {

	newJamName := fmt.Sprintf("user_appdata$%s", jamName)

	err := client.CreateDB(context.TODO(), newJamName)
	if err != nil {
		return nil, err
	}

	newJamDB := client.DB(newJamName)
	_, err = newJamDB.Put(context.TODO(), "_design/membership", map[string]interface{}{
		"_id": "_design/membership",
		"views": map[string]interface{}{
			"getMembership": map[string]interface{}{
				"map": "function (doc) {\n  if (doc.type == 'Member' || doc.type == 'Band') {\n    emit(doc.join_date_iso, doc.type);\n  }\n}",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = newJamDB.Put(context.TODO(), "_design/types", map[string]interface{}{
		"_id": "_design/types",
		"views": map[string]interface{}{
			"chatsByCreateTime": map[string]interface{}{
				"map": "function (doc) {\nif (doc.type == 'ChatMessage') {\n  emit(doc.created, null);\n  }\n}",
			},
			"loopsByCreateTime": map[string]interface{}{
				"map": "function (doc) {\nif (doc.type == 'Loop') {\n  emit(doc.created, null);\n  }\n}",
			},
			"rifffLoopsByCreateTime": map[string]interface{}{
				"map": "function (doc) {\nif (doc.type == 'Rifff') {\n  const array = [];\n  for (let slotNumber = 0; slotNumber < 8; slotNumber++) {\n    const playbackSlot = doc.state.playback[slotNumber].slot;\n    const currentEngine = playbackSlot.current;\n    if (currentEngine != null) {\n      const loopId = currentEngine.currentLoop;\n      if (loopId != null && loopId != '00000000000000000000000000000000') {\n        array.push(loopId);\n      }\n    }\n  }\n  emit(doc.created, array);\n}\n}",
			},
			"rifffsByCreateTime": map[string]interface{}{
				"map": "function (doc) {\nif (doc.type == 'Rifff') {\n  emit(doc.created, null);\n  }\n}",
			},
			"tracksByCreateTime": map[string]interface{}{
				"map": "function (doc) {\nif (doc.type == 'Track') {\n  emit(doc.created, null);\n  }\n}",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return newJamDB, nil
}

// -----------------------------------------------------------------------------------------------------------------------------------
// call something on a DB to see if it errors out with a particular failure string, indicating the database doesn't exist
func doesDatabaseExist(couchClient *kivik.Client, databaseName string) (bool, error) {

	dbRef := couchClient.DB(databaseName)
	_, dbExistErr := dbRef.Stats(context.TODO())

	// no error, which means (i assume) we're good
	if dbExistErr == nil {
		return true, nil
	}
	// very specific error return
	if strings.Contains(dbExistErr.Error(), "Not Found: Database does not exist") {
		return false, nil
	}

	// .. some unknown error condition
	return false, dbExistErr
}

// wrapper around doesDatabaseExist that formats to `user_appdata:<name>` to check for jam databases specifically
func doesJamDatabaseExist(couchClient *kivik.Client, jamName string) (bool, error) {
	return doesDatabaseExist(couchClient, fmt.Sprintf("user_appdata$%s", jamName))
}

// -----------------------------------------------------------------------------------------------------------------------------------
func createDefaultPublicJamDatabase(couchClient *kivik.Client, jamName string) error {

	newJamDB, err := createNewJamDatabase(couchClient, jamName)
	if err != nil {
		return fmt.Errorf("failed to create new jam database: %s", err.Error())
	}

	// snag the security block to configure it for default access
	soloSecurity, err := newJamDB.Security(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to acquire jam database security: %s", err.Error())
	}

	soloSecurity.Members.Roles = append(soloSecurity.Members.Names, "jammers")

	// write it back
	err = newJamDB.SetSecurity(context.TODO(), soloSecurity)
	if err != nil {
		return fmt.Errorf("failed to reconfigure jam database security: %s", err.Error())
	}

	// setup a default profile document; this will be updated for jams automatically on server boot from the jam manifest spec
	var defaultProfile JamDatabaseProfileData
	defaultProfile.Created = time.Now().UnixMilli()
	defaultProfile.DisplayName = jamName
	defaultProfile.Type = "Profile"

	_, err = newJamDB.Put(context.TODO(), "Profile", defaultProfile)
	if err != nil {
		return fmt.Errorf("unable to set jam Profile document: %s", err.Error())
	}

	return nil
}

// -----------------------------------------------------------------------------------------------------------------------------------
// https://docs.couchdb.org/en/stable/intro/security.html#org-couchdb-user
func getCouchRecordIDForUser(username string) string {
	return fmt.Sprintf("org.couchdb.user:%s", username)
}

// -----------------------------------------------------------------------------------------------------------------------------------

type UserExtra struct {
	Login string `json:"login"`
	Bio   string `json:"bio"`
}

// we stash some extra data in the _users database, this returns those fields
func fetchUserExtrasFromCouch(client *kivik.Client, username string) (*UserExtra, error) {

	userDb := client.DB("_users")
	var userEx UserExtra
	err := userDb.Get(context.TODO(), getCouchRecordIDForUser(username)).ScanDoc(&userEx)
	if err != nil {
		return nil, err
	}

	return &userEx, nil
}

// -----------------------------------------------------------------------------------------------------------------------------------
// produce a deterministic password for a user used for their personal Couch database access
// TODO: bcrypt or whatever. this dumb hashing was just a test
func generateInternalCouchUserPassword(username string) string {

	defaultSalt := "_salt"
	if viper.IsSet(cConfigCouchSalt) {
		defaultSalt = viper.GetString(cConfigCouchSalt)
	}

	byteInput := []byte(username + defaultSalt)
	md5Hash := md5.Sum(byteInput)
	return hex.EncodeToString(md5Hash[:])
}
