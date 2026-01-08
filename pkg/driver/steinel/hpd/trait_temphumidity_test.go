package hpd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-api/go/types"
)

func TestTemperatureSensor_GetAirTemperature(t *testing.T) {
	password := "Steinel123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Basic OlN0ZWluZWwxMjM=" {
			t.Errorf("incorrect Authorization header %q", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.RequestURI != "/rest/sensor" {
			t.Errorf("incorrect path requested %q", r.RequestURI)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_, _ = w.Write([]byte(`{"Temperature": 20.2, "Humidity": 50}`))
	}))

	baseURL, err := url.Parse(server.URL + "/rest")
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{
		BaseURL:  *baseURL,
		Client:   server.Client(),
		Password: password,
	}
	tempSensor := newTemperatureSensor(client, nil)

	res, err := tempSensor.GetAirTemperature(context.Background(), &traits.GetAirTemperatureRequest{})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	expect := &traits.AirTemperature{
		AmbientTemperature: &types.Temperature{
			ValueCelsius: 20.2,
		},
		AmbientHumidity: ref[float32](50.0),
	}
	if diff := cmp.Diff(expect, res, protocmp.Transform()); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
}

func ref[T any](t T) *T {
	return &t
}
