package iothub

import (
	"context"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
)

func MQTTClientOptions(hostName, deviceID, moduleID string, signer auth.Signer) (*mqtt.ClientOptions, error) {
	gen := func() (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return auth.GenerateSASToken(
			ctx, signer,
			auth.FormatSASURI(hostName, deviceID, moduleID),
			"",
			time.Now().Add(time.Hour),
		)
	}

	// Before proceeding, let's check we can actually generate a valid signature.
	// It's harder to report this problem to the user from the SetCredentialsProvider callback.
	// But if it succeeds here it will almost certainly succeed there too.
	_, err := gen()
	if err != nil {
		return nil, err
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(fmt.Sprintf("tls://%s:8883", hostName))
	options.SetClientID(deviceID)
	options.SetCredentialsProvider(func() (username string, password string) {
		username = fmt.Sprintf("%s/%s/?api-version=%s", hostName, deviceID, APIVersion)
		var err error
		password, err = gen()
		if err != nil {
			log.Printf("unable to generate SAS token: %v", err)
			return "", ""
		}
		return
	})
	return options, nil
}
