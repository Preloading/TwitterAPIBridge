package notifications

import (
	"fmt"
	"slices"

	blueskyapi "github.com/Preloading/TwitterAPIBridge/bluesky"
	"golang.org/x/net/websocket"
)

type JetstreamCommit struct {
	Operation string
	Record    blueskyapi.PostRecord
}

// Tailored for post because lazy
type JetstreamPostOutput struct {
	DID string `json:"did"`
	// Not all this data is needed, and it speeds up json decodes :)
	// TimeUS int64  `json:"time_us"`
	// Kind   string `json:"kind"`
	Commit JetstreamCommit `json:"commit"`
}

var (
	mentionedDIDs []string
	posterDIDs    []string
)

func RunNotifications() {
	ws, err := websocket.Dial("wss://jetstream1.us-east.bsky.network/subscribe?wantedCollections=app.bsky.feed.post", "", "https://jetstream1.us-east.bsky.network/subscribe?wantedCollections=app.bsky.feed.post")
	if err != nil {
		return
	}

	posterDIDs = append(posterDIDs, "did:plc:kpxax3hceauvwzagmux7gjuo")       // unit test acc #1
	mentionedDIDs = append(mentionedDIDs, "did:plc:jvioqrseaeiq42pz2jfwtuyg") // unit test acc #2
	mentionedDIDs = append(mentionedDIDs, "did:plc:khcyntihpu7snjszuojjgjc4") // preloading.dev

	incomingMessages := make(chan JetstreamPostOutput)
	go readJetstreamMessages(ws, incomingMessages)

	for message := range incomingMessages {
		if message.Commit.Operation != "create" {
			continue
		}
		if slices.Contains(posterDIDs, message.DID) {
			fmt.Printf("New Jetstream Message (user): %+v\n", message)
		}
		record := message.Commit.Record
		for _, facet := range record.Facets {
			for _, feature := range facet.Features {
				if feature.Type == "app.bsky.richtext.facet#mention" {
					// its got a mention

					// check if the mention is in our list of people subscribed to mention notifications
					if slices.Contains(mentionedDIDs, feature.Did) {
						fmt.Printf("New Jetstream Message (mention): %+v\n", record)
					}
				}
			}
		}
	}
}

func readJetstreamMessages(ws *websocket.Conn, incomingMessages chan JetstreamPostOutput) {
	for {
		var message JetstreamPostOutput
		err := websocket.JSON.Receive(ws, &message)
		// err := websocket.Message.Receive(ws, &message)
		if err != nil {
			fmt.Printf("Error::: %s\n", err.Error())
			return
		}
		incomingMessages <- message
	}
}

// This function is quite a bit slower than our inital check, and it does the following:
// 1. ~~If it's a mention, verify that the user hasn't blocked~~ I dont think this is possible.
// 2. Get the device tokens of the devices that would like these specific push notifications.
// 3. Converting the text into a twitter post
// 4. Send the twitter post's content as a push notification via SGN.
func sendPushNotification() {

}
