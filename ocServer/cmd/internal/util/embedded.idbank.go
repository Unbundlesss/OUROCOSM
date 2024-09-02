//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//
// -----------------------------------------------------------------------------------------------------------------------------------
//
// Endlesss jams are known by various IDs - the original is the suffix used in the couchDB to denote the jam's own database,
// these look like `band##########` where the #s are a 10-digit hex number, unique to the platform. We'll call these Couch IDs.
//
// Another ID used would be the invite/join-link or listen-link ID which is an encrypted form of the Couch ID. They are used
// where the platform doesn't want to reveal the original database ID, they are a longer hex stream, for example:
// "5a157ceaf13a7fc6208661626a03bdb5b97d0d126fdda675bfa2f0f1269b77d7"
// 						... is the encrypted form of "band2f1f1597b0"
//
// We do not have the cryptographic keys or method used to move between IDs, so unfortunately we have to scavenge existing pairs
// from the original Endlesss service to re-use for our own purposes. Luckily, I happened to save off over 900 of those pairs as
// part of the data-scraping for OUROVEON LORE, so that should keep us in IDs for a while:
// https://github.com/Unbundlesss/OUROVEON/blob/main/bin/shared/endlesss.publics.json
//
// In COSM, we can therefore now refer to jams by a simpler incremental title, eg. "Jam_001" and the code below will
// be responsible for translating that into Endlesssian space for us. See the associated .json for details.
//
// The band IDs embedded here are taken from the above LORE dataset. As that guarantees overlap with previous Endlesss jams, the export tools
// built into ocServer define a new band##### export signature to differentiate them (see op.export.go)
//

package util

import (
	_ "embed"
	"encoding/json"
)

//go:embed embedded.idbank.json
var GlobalIDBankJson []byte

type JamID struct {
	CouchID string `json:"couch"`
	LongID  string `json:"encrypted"`
}
type JamIDBank struct {
	Entries map[string]JamID `json:"entries"`
}

// -----------------------------------------------------------------------------------------------------------------------------------

type JamIDs struct {
	global        JamIDBank
	couchToLong   map[string]string
	couchToCosmid map[string]string
	longToCouch   map[string]string
}

func (jids JamIDs) Bank() JamIDBank {
	return jids.global
}

func (jids JamIDs) CouchFromLong(longID string) (string, bool) {
	result, ok := jids.longToCouch[longID]
	return result, ok
}

func (jids JamIDs) LongFromCouch(couchID string) (string, bool) {
	result, ok := jids.couchToLong[couchID]
	return result, ok
}

func (jids JamIDs) CosmidFromCouch(couchID string) (string, bool) {
	result, ok := jids.couchToCosmid[couchID]
	return result, ok
}

// -----------------------------------------------------------------------------------------------------------------------------------
// parse the embedded JSON, deconstruct into maps for use by the app code
func LoadJamIDBanks() (*JamIDs, error) {

	result := JamIDs{}

	err := json.Unmarshal(GlobalIDBankJson, &result.global)
	if err != nil {
		return nil, err
	}

	result.couchToLong = make(map[string]string)
	result.couchToCosmid = make(map[string]string)
	result.longToCouch = make(map[string]string)

	for k, v := range result.global.Entries {

		// create additional convenience lookups to move between IDs
		result.couchToLong[v.CouchID] = v.LongID
		result.couchToCosmid[v.CouchID] = k
		result.longToCouch[v.LongID] = v.CouchID
	}

	return &result, nil
}
