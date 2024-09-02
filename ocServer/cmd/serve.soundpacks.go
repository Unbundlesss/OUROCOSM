//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------------------------------------------------------------
// format of the returned sampler manifest via GET /sound-packs
type SamplerSoundPack struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Creator         string    `json:"creator"`
	Created         int64     `json:"created"`
	CreatedIso      time.Time `json:"createdIso"`
	LastModified    int64     `json:"lastModified"`
	LastModifiedIso time.Time `json:"lastModifiedIso"`
	Presets         []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		SampleUrls  []struct {
			Mime     string `json:"mime"`
			URL      string `json:"url"`
			Hash     string `json:"hash"`
			Length   int    `json:"length"`
			Endpoint string `json:"endpoint"`
			Key      string `json:"key"`
			Name     string `json:"name"`
		} `json:"sampleUrls"`
		SamplerState    string    `json:"samplerState"`
		Created         int64     `json:"created"`
		CreatedIso      time.Time `json:"createdIso"`
		LastModified    int64     `json:"lastModified"`
		LastModifiedIso time.Time `json:"lastModifiedIso"`
		State           string    `json:"state"`
	} `json:"presets"`
}
type SamplerData struct {
	SoundPacks []SamplerSoundPack `json:"soundPacks"`
}
type SamplerResponseData struct {
	Ok   bool        `json:"ok"`
	Data SamplerData `json:"data"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
// NB if we don't return the bare minimum response JSON here, Studio will immediately crash
// bare minimum means:
// {"ok":true,"data":{"soundPacks":[]}}
func HandlerSoundPacksGet(httpResponse http.ResponseWriter, r *http.Request) {

	//authUsername, _, err := decodeAccountAuthBearer(r)
	//if err != nil {
	//	http.Error(httpResponse, err.Error(), http.StatusForbidden)
	//	return
	//}

	soundPacks := make([]SamplerSoundPack, 0, 10)
	samplerData := SamplerData{
		SoundPacks: soundPacks,
	}
	samplerResponse := &SamplerResponseData{
		Ok:   true,
		Data: samplerData,
	}

	//jsonData, _ := json.Marshal(samplerResponse)
	//fmt.Printf("SP GET: %s\n%s", authUsername, jsonData)

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(samplerResponse)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// POST /sound-packs
// usually something like '{"name": "My Sounds", "description": ""}', the top level group for all sub-packs(?)

type SoundPackRoot struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func HandlerSoundPacksPost(httpResponse http.ResponseWriter, r *http.Request) {

	authUsername, _, err := decodeAccountAuthBearer(r)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusForbidden)
		return
	}

	var soundPackRoot SoundPackRoot
	err = json.NewDecoder(r.Body).Decode(&soundPackRoot)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusBadRequest)
		return
	}

	SysLog.Info("SoundPack root request", zap.String("User", authUsername), zap.String("Pack", soundPackRoot.Name))
	//for k, v := range r.Header {
	//	fmt.Printf("  [%s] = %s\n", k, v)
	//}

	httpResponse.WriteHeader(http.StatusOK)
}

// -----------------------------------------------------------------------------------------------------------------------------------
type SoundPackUpdate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SampleUrls  []struct {
		Mime     string `json:"mime"`
		URL      string `json:"url"`
		Hash     string `json:"hash"`
		Length   int    `json:"length"`
		Endpoint string `json:"endpoint"`
		Key      string `json:"key"`
		Name     string `json:"name"`
	} `json:"sampleUrls"`
	SamplerState string `json:"samplerState"`
}

// POST /sound-packs/presets
func HandlerSoundPacksPresetsPost(httpResponse http.ResponseWriter, r *http.Request) {

	authUsername, _, err := decodeAccountAuthBearer(r)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusForbidden)
		return
	}

	var soundPackUpdate SoundPackUpdate
	err = json.NewDecoder(r.Body).Decode(&soundPackUpdate)
	if err != nil {
		http.Error(httpResponse, err.Error(), http.StatusBadRequest)
		return
	}

	SysLog.Info("SoundPack update", zap.String("User", authUsername))

	//debugOut, _ := json.MarshalIndent(soundPackUpdate, "", "    ")
	//fmt.Printf("%s\n", string(debugOut))

	httpResponse.WriteHeader(http.StatusOK)
}
