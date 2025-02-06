//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

// -----------------------------------------------------------------------------------------------------------------------------------
// riff metadata as per couch
type JamRiffData struct {
	ID    string `json:"_id"`
	Rev   string `json:"_rev"`
	State struct {
		BarLength int     `json:"barLength"`
		Bps       float64 `json:"bps"`
		Playback  []struct {
			// this is always empty / useless data so we skip it
			//			Effects struct {
			//				Slots []struct {
			//					Current any `json:"current"`
			//				} `json:"slots"`
			//			} `json:"effects"`
			Slot struct {
				Current struct {
					On          bool    `json:"on"`
					Type        string  `json:"type"`
					CurrentLoop string  `json:"currentLoop"`
					Gain        float64 `json:"gain"`
				} `json:"current"`
			} `json:"slot"`
		} `json:"playback"`
	} `json:"state"`
	Scale        int       `json:"scale"`
	AppVersion   int       `json:"app_version"`
	Type         string    `json:"type"`
	SentBy       string    `json:"sentBy"`
	UserName     string    `json:"userName"`
	Colour       string    `json:"colour"`
	LayerColours []string  `json:"layerColours"`
	Magnitude    float64   `json:"magnitude"`
	Created      int64     `json:"created"`
	Root         int       `json:"root"`
	Brightness   float64   `json:"brightness"`
	PeakData     []float64 `json:"peakData"`
}

type EndpointAudio struct {
	Endpoint string `json:"endpoint"`
	Hash     string `json:"hash"`
	Key      string `json:"key"`
	Length   int    `json:"length"`
	Mime     string `json:"mime"`
	URL      string `json:"url"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
// stem / loop data as per couch
type JamStemData struct {
	ID             string  `json:"_id"`
	Rev            string  `json:"_rev"`
	AppVersion     int     `json:"app_version"`
	BarLength      int     `json:"barLength"`
	Bps            float64 `json:"bps"`
	CdnAttachments struct {
		OggAudio  EndpointAudio `json:"oggAudio"`
		FlacAudio EndpointAudio `json:"flacAudio"`
	} `json:"cdn_attachments"`
	ColourHistory       []string `json:"colourHistory"`
	Created             int64    `json:"created"`
	CreatorUserName     string   `json:"creatorUserName"`
	IsBass              bool     `json:"isBass"`
	IsDrum              bool     `json:"isDrum"`
	IsMic               bool     `json:"isMic"`
	IsNormalised        bool     `json:"isNormalised"`
	IsNote              bool     `json:"isNote"`
	Length              int      `json:"length"`
	Length16Ths         int      `json:"length16ths"`
	MaxAllowedPeakLevel float64  `json:"maxAllowedPeakLevel"`
	OriginalPitch       float64  `json:"originalPitch"`
	PeakLevel           float64  `json:"peakLevel"`
	PresetName          string   `json:"presetName"`
	PrimaryColour       string   `json:"primaryColour"`
	SampleRate          float64  `json:"sampleRate"`
	Type                string   `json:"type"`
}

func getActiveEndpoint(stemData JamStemData) *EndpointAudio {
	if len(stemData.CdnAttachments.FlacAudio.Endpoint) > 0 {
		return &stemData.CdnAttachments.FlacAudio
	}
	return &stemData.CdnAttachments.OggAudio
}
