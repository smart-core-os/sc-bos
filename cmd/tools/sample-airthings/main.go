// The sample application uses the AirThings API to pull sensor readings (latest samples) from a named location.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"

	"github.com/smart-core-os/sc-bos/pkg/driver/airthings/api"
)

func main() {
	var clientID, clientSecret string
	var locationName string
	flag.StringVar(&clientID, "client-id", "", "client id")
	flag.StringVar(&clientSecret, "client-secret", "", "client secret")
	flag.StringVar(&locationName, "location", "", "location name")
	flag.Parse()

	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://accounts-api.airthings.com/v1/token",
		Scopes:       []string{"read:device"},
	}
	ctx := context.Background()
	client := conf.Client(ctx)

	var locations api.GetLocationsResponse
	err := get(client, "https://ext-api.airthings.com/v1/locations", &locations)
	if err != nil {
		log.Fatalf("list locations %v", err)
	}

	var ew api.LocationResponse
	for _, location := range locations.GetLocations() {
		if location.Name == locationName {
			ew = location
			break
		}
	}
	if ew.Id == "" {
		var locationNames []string
		for _, location := range locations.GetLocations() {
			locationNames = append(locationNames, location.Name)
		}
		log.Fatalf("Failed to find %q, available locations: %q", locationName, locationNames)
	}

	var samples api.GetLocationSamplesResponseEnriched
	err = get(client, fmt.Sprintf("https://ext-api.airthings.com/v1/locations/%v/latest-samples", ew.Id), &samples)
	if err != nil {
		log.Fatalf("list samples %v", err)
	}

	for _, device := range samples.GetDevices() {
		fmt.Printf("%#v\n", device)
	}
}

func get(client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request %w", err)
	}
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do get %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("bad response %v %v, failed to read body %v", res.StatusCode, res.Status, err)
		}
		return fmt.Errorf("bad response %v %v\n%v", res.StatusCode, res.Status, string(body))
	}
	return json.NewDecoder(res.Body).Decode(v)
}
