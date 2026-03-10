package server

import (
	"context"
	"sync"

	"github.com/smart-core-os/sc-api/go/info"
	"go.uber.org/zap"
)

type InfoServer struct {
	info.UnimplementedInfoServer
	deviceMapSync struct {
		sync.RWMutex
		m map[string]*info.Device
	}
	logger *zap.Logger
}

func NewInfoServer(logger *zap.Logger) *InfoServer {
	return &InfoServer{
		deviceMapSync: struct {
			sync.RWMutex
			m map[string]*info.Device
		}{m: make(map[string]*info.Device)},
		logger: logger,
	}
}

func (i *InfoServer) Shutdown() {

}

func (i *InfoServer) AddDevice(device *info.Device) bool {
	i.deviceMapSync.Lock()
	_, exists := i.deviceMapSync.m[device.Name]
	if exists {
		i.deviceMapSync.Unlock()
		return false
	}

	i.deviceMapSync.m[device.Name] = device
	i.deviceMapSync.Unlock()
	i.logger.Debug("Device added successfully", zap.String("device", device.String()))
	return true
}

func (i *InfoServer) RemoveDevice(device *info.Device) bool {
	i.deviceMapSync.Lock()
	_, exists := i.deviceMapSync.m[device.Name]
	if exists {
		i.deviceMapSync.Unlock()
		return false
	}
	delete(i.deviceMapSync.m, device.Name)
	i.deviceMapSync.Unlock()
	i.logger.Debug("Device removed successfully", zap.String("device name", device.Name))
	return true
}

func (i *InfoServer) ListDevices(ctx context.Context, request *info.ListDevicesRequest) (*info.ListDevicesResponse, error) {
	var d []*info.Device
	i.deviceMapSync.RLock()
	for _, dev := range i.deviceMapSync.m {
		d = append(d, dev)
	}
	i.deviceMapSync.RUnlock()
	return &info.ListDevicesResponse{
		Devices:       d,
		NextPageToken: "",
		TotalSize:     0,
	}, nil
}
