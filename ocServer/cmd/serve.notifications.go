//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"net/http"
)

type NotificationsData struct {
	DummyNote []string `json:"notifications"`
}
type NotificationsResponse struct {
	Okay bool              `json:"ok"`
	Data NotificationsData `json:"data"`
}

// -----------------------------------------------------------------------------------------------------------------------------------
// currently Notification feed is unsupported
func HandlerNotifications(httpResponse http.ResponseWriter, r *http.Request) {

	mySubscriptionResponse := &NotificationsResponse{
		Okay: true,
		Data: NotificationsData{[]string{}},
	}

	httpResponse.Header().Set(HeaderNameContentType, ContentTypeApplicationJson)
	httpResponse.WriteHeader(http.StatusOK)
	json.NewEncoder(httpResponse).Encode(mySubscriptionResponse)
}

// -----------------------------------------------------------------------------------------------------------------------------------
// don't actually have a recorded trace for these; they are generated when a riff arrives in a jam and you might
// want to bubble out a notification to the affected user to let them know activity is happening
//
// I'd say we could capture and add some Notifications to return to people but the Notifications UI in Studio kind of sucks so
// maybe best to just leave it alone
func HandlerNotifyData(httpResponse http.ResponseWriter, r *http.Request) {

	httpResponse.WriteHeader(http.StatusOK)
}
