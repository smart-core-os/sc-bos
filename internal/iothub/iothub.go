// Package iothub provides a client for Azure IoT Hub.
//
// Connect to an IoT Hub using [Dial] passing in a [ConnectionParameters] struct.
// [ConnectionParameters] can be created manually, parsed from a connection string using [ParseConnectionString],
// or created dynamically using the Device Provisioning Service using the [github.com/smart-core-os/sc-bos/pkg/internal/iothub/dps] package.
package iothub

const APIVersion = "2021-04-12"
