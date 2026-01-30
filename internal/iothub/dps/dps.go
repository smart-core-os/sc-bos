// Package dps implements provisioning a device using the Azure IoT Device Provisioning Service.
// Only flows for SAS authentication over HTTPS are implemented.
// Entry point is calling Register.
//
// Based on the Azure IoT docs at https://learn.microsoft.com/en-us/azure/iot/iot-mqtt-connect-to-iot-dps
package dps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/smart-core-os/sc-bos/internal/iothub"
	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
)

const (
	DefaultHost = "global.azure-devices-provisioning.net"
	APIVersion  = "2019-03-31"
)

const (
	registrationPolicyName = "registration"
	apiVersionQueryString  = "api-version=" + APIVersion
	opStatusAssigned       = "assigned"
	opStatusDisabled       = "disabled"
)

type Client interface {
	// Register will complete a
	Register(ctx context.Context, registrationID string) (Registration, error)
	io.Closer
}

type Registration struct {
	HostName        string // Domain name of IoT Hub to connect to.
	DeviceID        string
	SharedAccessKey auth.SASKey
}

func (r Registration) AsConnectionParameters() iothub.ConnectionParameters {
	return iothub.ConnectionParameters{
		HostName:        r.HostName,
		SharedAccessKey: r.SharedAccessKey,
		DeviceID:        r.DeviceID,
	}
}

func FormatSASURI(idScope, registrationID string) string {
	return fmt.Sprintf("%s/registrations/%s", idScope, registrationID)
}

var ErrEnrollmentDisabled = errors.New("enrollment is disabled")

// Register will attempt to register a device with the device provisioning service.
//
//   - host is the hostname of the Device Provisioning Service. Defaults to DefaultHost if empty.
//   - idScope identifies the instance of device provisioning service to register the device in. Required.
//   - regID is an identifier for the registration, unique within an idScope. If using Group Enrollments, then the
//     Registration ID becomes the new Device ID.
//   - key is the shared secret used to authenticate the device. Required.
func Register(ctx context.Context, host, idScope, regID string, key auth.SASKey) (Registration, error) {
	if host == "" {
		host = DefaultHost
	}
	if idScope == "" || regID == "" {
		return Registration{}, errors.New("idScope and regID are required")
	}

	token, err := auth.GenerateSASToken(ctx,
		&auth.LocalSigner{Secret: key},
		FormatSASURI(idScope, regID),
		registrationPolicyName,
		time.Now().Add(time.Hour),
	)
	if err != nil {
		return Registration{}, err
	}

	req := putRegistrationRequest{
		RegistrationID: regID,
	}
	var putRes putRegistrationResponse
	header, err := httpPutJSON(ctx, putRegistrationEndpoint(host, idScope, regID).String(), token, req, &putRes)
	if err != nil {
		return Registration{}, err
	}
	if putRes.Status == opStatusDisabled {
		return Registration{}, ErrEnrollmentDisabled
	}
	log.Printf("registration %q started with operation %q", regID, putRes.OperationID)

	for {
		after, err := retryAfter(header)
		if err != nil {
			log.Printf("can't work out when to poll - assuming 10s: %s", err.Error())
			after = 10 * time.Second
		}

		select {
		case <-time.After(after):
		case <-ctx.Done():
			return Registration{}, ctx.Err()
		}

		var opStatus operationStatus
		header, err = httpGetJSON(ctx,
			getOperationEndpoint(host, idScope, regID, putRes.OperationID).String(),
			token, &opStatus)
		if err != nil {
			return Registration{}, err
		}

		log.Printf("operation %q status is now %q", putRes.OperationID, opStatus.Status)
		if opStatus.Status == opStatusAssigned {
			return Registration{
				HostName:        opStatus.RegistrationState.AssignedHub,
				DeviceID:        opStatus.RegistrationState.DeviceID,
				SharedAccessKey: key,
			}, nil
		}
	}

}

func baseEndpoint(host, idScope, regID string) *url.URL {
	return (&url.URL{Scheme: "https", Host: host}).JoinPath(idScope, "registrations", regID)
}

func putRegistrationEndpoint(host, idScope, regID string) *url.URL {
	u := baseEndpoint(host, idScope, regID).JoinPath("register")
	u.RawQuery = apiVersionQueryString
	return u
}

func getOperationEndpoint(host, idScope, regID, opID string) *url.URL {
	u := baseEndpoint(host, idScope, regID).JoinPath("operations", opID)
	u.RawQuery = apiVersionQueryString
	return u
}

func retryAfter(header http.Header) (time.Duration, error) {
	retryAfterStr := header.Get("retry-after")
	seconds, err := strconv.ParseFloat(retryAfterStr, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(float64(time.Second) * seconds), nil
}

type putRegistrationRequest struct {
	RegistrationID string `json:"registrationId"`
}

type putRegistrationResponse struct {
	Status      string `json:"status"`
	OperationID string `json:"operationId"`
}

type operationStatus struct {
	OperationID       string             `json:"operationId"`
	Status            string             `json:"status"`
	RegistrationState *registrationState `json:"registrationState"`
}

type registrationState struct {
	RegistrationID         string    `json:"registrationId"`
	CreatedDateTimeUTC     time.Time `json:"createdDateTimeUtc"`
	AssignedHub            string    `json:"assignedHub"`
	DeviceID               string    `json:"deviceId"`
	Status                 string    `json:"status"`
	SubStatus              string    `json:"substatus"`
	LastUpdatedDateTimeUTC time.Time `json:"lastUpdatedDateTimeUtc"`
	ETag                   string    `json:"etag"`
}
