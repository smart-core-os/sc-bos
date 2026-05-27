package sc

// The request orchestration below mirrors gobacnet's Client methods (request.go,
// readmulti.go, writeprop.go, whois.go, remote.go, objectlist.go), substituting
// the websocket datalink for gobacnet's UDP send and this package's tsm/utsm for
// gobacnet's. It is derivative of gobacnet (GPLv2 with linking exception).

import (
	"context"
	"errors"
	"fmt"

	"github.com/smart-core-os/gobacnet/encoding"
	"github.com/smart-core-os/gobacnet/enum/errorcode"
	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
	"github.com/smart-core-os/gobacnet/types/objecttype"
)

// ReadProperty reads a single property from a single object on dest.
func (c *Client) ReadProperty(ctx context.Context, dest bactypes.Device, rp bactypes.ReadPropertyData) (bactypes.ReadPropertyData, error) {
	id, err := c.tsm.ID(ctx)
	if err != nil {
		return bactypes.ReadPropertyData{}, fmt.Errorf("unable to get a transaction id: %w", err)
	}
	defer c.tsm.Put(id)

	enc := encoding.NewEncoder()
	enc.NPDU(requestNPDU(dest.Addr))
	enc.ReadProperty(uint8(id), rp)
	if enc.Error() != nil {
		return bactypes.ReadPropertyData{}, fmt.Errorf("encode read property: %w", enc.Error())
	}

	var out bactypes.ReadPropertyData
	if err := c.link.Send(dest.Addr, enc.Bytes()); err != nil {
		return out, err
	}
	raw, err := c.tsm.Receive(ctx, id)
	if err != nil {
		return out, fmt.Errorf("unable to receive id %d: %w", id, err)
	}
	switch v := raw.(type) {
	case error:
		return out, v
	case []byte:
		dec := encoding.NewDecoder(v)
		var apdu bactypes.APDU
		dec.APDU(&apdu)
		dec.ReadProperty(&out)
		return out, dec.Error()
	default:
		return out, fmt.Errorf("received unknown datatype %T", raw)
	}
}

// ReadMultiProperty reads multiple properties from dev in a single request.
func (c *Client) ReadMultiProperty(ctx context.Context, dev bactypes.Device, rp bactypes.ReadMultipleProperty) (bactypes.ReadMultipleProperty, error) {
	var out bactypes.ReadMultipleProperty

	id, err := c.tsm.ID(ctx)
	if err != nil {
		return out, fmt.Errorf("unable to get transaction id: %w", err)
	}
	defer c.tsm.Put(id)

	enc := encoding.NewEncoder()
	enc.NPDU(requestNPDU(dev.Addr))
	if err := enc.ReadMultipleProperty(uint8(id), rp); err != nil {
		return out, fmt.Errorf("encode read multiple property: %w", err)
	}
	pack := enc.Bytes()
	if dev.MaxApdu != 0 && dev.MaxApdu < uint32(len(pack)) {
		return out, fmt.Errorf("read multiple property is too large (max: %d given: %d)", dev.MaxApdu, len(pack))
	}

	if err := c.link.Send(dev.Addr, pack); err != nil {
		return out, err
	}
	raw, err := c.tsm.Receive(ctx, id)
	if err != nil {
		return out, fmt.Errorf("unable to receive id %d: %w", id, err)
	}
	switch v := raw.(type) {
	case error:
		return out, v
	case []byte:
		dec := encoding.NewDecoder(v)
		var apdu bactypes.APDU
		dec.APDU(&apdu)
		if err := dec.ReadMultiplePropertyAck(&out); err != nil {
			return out, err
		}
		return out, nil
	default:
		return out, fmt.Errorf("received unknown datatype %T", raw)
	}
}

// WriteProperty writes a single property to a single object on dest.
func (c *Client) WriteProperty(ctx context.Context, dest bactypes.Device, wp bactypes.ReadPropertyData, priority uint) error {
	id, err := c.tsm.ID(ctx)
	if err != nil {
		return fmt.Errorf("unable to get a transaction id: %w", err)
	}
	defer c.tsm.Put(id)

	enc := encoding.NewEncoder()
	enc.NPDU(requestNPDU(dest.Addr))
	enc.WriteProperty(uint8(id), wp, priority)
	if enc.Error() != nil {
		return fmt.Errorf("encode write property: %w", enc.Error())
	}

	if err := c.link.Send(dest.Addr, enc.Bytes()); err != nil {
		return err
	}
	raw, err := c.tsm.Receive(ctx, id)
	if err != nil {
		return fmt.Errorf("unable to receive id %d: %w", id, err)
	}
	switch v := raw.(type) {
	case error:
		return v
	case []byte:
		dec := encoding.NewDecoder(v)
		var apdu bactypes.APDU
		dec.APDU(&apdu)
		return dec.Error()
	default:
		return fmt.Errorf("received unknown datatype %T", raw)
	}
}

// WhoIs broadcasts a Who-Is to the hub and collects I-Am responses with device
// instance ids in [low, high]. Pass -1 for both to scan the whole network.
func (c *Client) WhoIs(ctx context.Context, low, high int) ([]bactypes.Device, error) {
	dest := bactypes.Address{Net: 0xFFFF}
	dest.SetBroadcast(true)

	enc := encoding.NewEncoder()
	enc.NPDU(bactypes.NPDU{
		Version:               bactypes.ProtocolVersion,
		Destination:           &dest,
		IsNetworkLayerMessage: false,
		ExpectingReply:        false,
		Priority:              bactypes.Normal,
		HopCount:              bactypes.DefaultHopCount,
	})
	if err := enc.WhoIs(int32(low), int32(high)); err != nil {
		return nil, err
	}

	start, end := 0, maxInt
	if low != -1 && high != -1 {
		start, end = low, high
	}

	errChan := make(chan error, 1)
	go func() { errChan <- c.link.Send(dest, enc.Bytes()) }()

	values, err := c.utsm.Subscribe(ctx, start, end)
	if err != nil {
		return nil, err
	}
	if err := <-errChan; err != nil {
		return nil, err
	}

	unique := make(map[bactypes.ObjectInstance]bactypes.Device)
	var devices []bactypes.Device
	for _, v := range values {
		iam, ok := v.(bactypes.IAm)
		if !ok {
			continue
		}
		if _, seen := unique[iam.ID.Instance]; seen {
			continue
		}
		dev := bactypes.Device{
			Addr:         iam.Addr,
			ID:           iam.ID,
			MaxApdu:      iam.MaxApdu,
			Segmentation: iam.Segmentation,
			Vendor:       iam.Vendor,
		}
		unique[iam.ID.Instance] = dev
		devices = append(devices, dev)
	}
	return devices, nil
}

// RemoteDevices reads device details for the given device instances at addr,
// returning data equivalent to an I-Am response. Mirrors gobacnet.RemoteDevices.
func (c *Client) RemoteDevices(ctx context.Context, addr bactypes.Address, ids ...bactypes.ObjectInstance) ([]bactypes.Device, error) {
	defaultDevice := bactypes.Device{Addr: addr, MaxApdu: 1000}
	req := bactypes.ReadMultipleProperty{}
	for _, id := range ids {
		oid := bactypes.ObjectID{Type: objecttype.Device, Instance: id}
		req.Objects = append(req.Objects, bactypes.Object{ID: oid, Properties: []bactypes.Property{
			{ID: property.MaxApduLengthAccepted, ArrayIndex: bactypes.ArrayAll},
			{ID: property.SegmentationSupported, ArrayIndex: bactypes.ArrayAll},
			{ID: property.VendorIdentifier, ArrayIndex: bactypes.ArrayAll},
		}})
	}
	res, err := c.readProperties(ctx, defaultDevice, req)
	if err != nil {
		return nil, err
	}
	devices := make([]bactypes.Device, len(res.Objects))
	for i, object := range res.Objects {
		if len(object.Properties) != 3 {
			return nil, fmt.Errorf("expected three object properties, got %d for %s", len(object.Properties), object.ID)
		}
		device := bactypes.Device{Addr: addr, ID: object.ID}
		maxApduProp, segProp, vendorProp := object.Properties[0], object.Properties[1], object.Properties[2]
		device.MaxApdu = maxApduProp.Data.(uint32)
		device.Segmentation = bactypes.Enumerated(segProp.Data.(uint32))
		device.Vendor = vendorProp.Data.(uint32)
		devices[i] = device
	}
	return devices, nil
}

// readProperties reads multiple properties, falling back to individual reads if
// the device rejects the combined request. Mirrors gobacnet.ReadProperties.
func (c *Client) readProperties(ctx context.Context, dev bactypes.Device, req bactypes.ReadMultipleProperty) (bactypes.ReadMultipleProperty, error) {
	res, err := c.ReadMultiProperty(ctx, dev, req)
	if err == nil {
		return res, nil
	}
	if ctx.Err() != nil {
		return res, err
	}
	if bacErr := (bactypes.Error{}); errors.As(err, &bacErr) {
		if !shouldRetryIndividually(bacErr.Code) {
			return bactypes.ReadMultipleProperty{}, err
		}
	}
	out := bactypes.ReadMultipleProperty{Objects: make([]bactypes.Object, len(req.Objects))}
	for i, object := range req.Objects {
		if ctx.Err() != nil {
			return bactypes.ReadMultipleProperty{}, ctx.Err()
		}
		propRes, err := c.ReadProperty(ctx, dev, bactypes.ReadPropertyData{Object: object})
		if err != nil {
			return bactypes.ReadMultipleProperty{}, err
		}
		out.Objects[i] = propRes.Object
	}
	return out, nil
}

func shouldRetryIndividually(code errorcode.ErrorCode) bool {
	switch code {
	case errorcode.OptionalFunctionalityNotSupported, errorcode.ServiceRequestDenied:
		return true
	case errorcode.AbortBufferOverflow, errorcode.AbortSegmentationNotSupported, errorcode.MessageTooLong, errorcode.AbortApduTooLong:
		return true
	case errorcode.DeviceBusy, errorcode.Timeout, errorcode.Busy, errorcode.AbortOutOfResources:
		return true
	default:
		return false
	}
}
