package iothub

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
)

type ConnectionParameters struct {
	HostName              string
	SharedAccessKeyName   string
	SharedAccessKey       auth.SASKey
	SharedAccessSignature string
	DeviceID              string
	ModuleID              string
	GatewayHostName       string
	X509                  string
}

func ParseConnectionString(cs string) (ConnectionParameters, error) {
	// Implementation adapted from https://github.com/Azure/azure-iot-sdk-python/blob/fa16df1452141452bff2bc800312d50817d6cbd6/azure-iot-device/azure/iot/device/common/auth/connection_string.py

	// parse key/value format
	segments := strings.Split(cs, csDelimiter)
	fields := make(map[string]string)
	for _, segment := range segments {
		key, value, ok := strings.Cut(segment, csSeparator)
		if !ok {
			return ConnectionParameters{}, ErrInvalidConnectionString
		}

		// prevent duplicate assignments
		if _, exists := fields[key]; exists {
			return ConnectionParameters{}, fmt.Errorf("%w: duplicate field %s", ErrInvalidConnectionString, key)
		}
		fields[key] = value
	}

	// by deleting each field as we read them, we know that any keys remaining at the end are unrecognised
	take := func(key string) string {
		value := fields[key] // missing fields read as "" intentionally
		delete(fields, key)
		return value
	}

	data := ConnectionParameters{
		HostName:              take(csKeyHostName),
		SharedAccessKeyName:   take(csKeySharedAccessKeyName),
		SharedAccessSignature: take(csKeySharedAccessSignature),
		DeviceID:              take(csKeyDeviceID),
		ModuleID:              take(csKeyModuleID),
		GatewayHostName:       take(csKeyGatewayHostName),
		X509:                  take(csKeyX509),
	}
	if keyStr := take(csKeySharedAccessKey); keyStr != "" {
		key, err := auth.ParseSASKey(keyStr)
		if err != nil {
			return ConnectionParameters{}, err
		}
		data.SharedAccessKey = key
	}
	// all fields should have been taken; any left are unrecognised; we will reject the connection string to be safe
	// as this is a security-critical context
	if keys := maps.Keys(fields); len(keys) > 0 {
		return ConnectionParameters{}, fmt.Errorf("%w: unknown field %s", ErrInvalidConnectionString, keys[0])
	}

	return data, nil
}

var ErrInvalidConnectionString = errors.New("invalid connection string")

const (
	csDelimiter = ";"
	csSeparator = "="

	csKeyHostName              = "HostName"
	csKeySharedAccessKeyName   = "SharedAccessKeyName"
	csKeySharedAccessKey       = "SharedAccessKey"
	csKeySharedAccessSignature = "SharedAccessSignature"
	csKeyDeviceID              = "DeviceId"
	csKeyModuleID              = "ModuleId"
	csKeyGatewayHostName       = "GatewayHostName"
	csKeyX509                  = "x509"
)
