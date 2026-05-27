package sc

// Object enumeration ported from gobacnet's objectlist.go (GPLv2 with linking
// exception). It builds a device's object list and enriches each object with its
// name and description, using this client's ReadProperty/ReadMultiProperty.

import (
	"context"
	"fmt"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"
)

const (
	readPropRequestSize   = 20
	maxStandardBacnetType = 128
)

// Objects retrieves all objects on dev, including each object's name and
// description, and returns dev with its Objects populated.
func (c *Client) Objects(ctx context.Context, dev bactypes.Device) (bactypes.Device, error) {
	if err := c.objectList(ctx, &dev); err != nil {
		return dev, fmt.Errorf("unable to get object list: %w", err)
	}
	if err := c.allObjectInformation(ctx, &dev); err != nil {
		return dev, fmt.Errorf("unable to get object information: %w", err)
	}
	return dev, nil
}

func (c *Client) objectListLen(ctx context.Context, dev bactypes.Device) (int, error) {
	rp := bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: dev.ID,
			Properties: []bactypes.Property{
				{ID: property.ObjectList, ArrayIndex: 0},
			},
		},
	}
	resp, err := c.ReadProperty(ctx, dev, rp)
	if err != nil {
		return 0, fmt.Errorf("reading property failed: %w", err)
	}
	if len(resp.Object.Properties) == 0 {
		return 0, fmt.Errorf("no data was returned")
	}
	data, ok := resp.Object.Properties[0].Data.(uint32)
	if !ok {
		return 0, fmt.Errorf("unable to get object length")
	}
	return int(data), nil
}

func (c *Client) objectsRange(ctx context.Context, dev bactypes.Device, start, end int) ([]bactypes.Object, error) {
	rpm := bactypes.ReadMultipleProperty{
		Objects: []bactypes.Object{{ID: dev.ID}},
	}
	for i := start; i <= end; i++ {
		rpm.Objects[0].Properties = append(rpm.Objects[0].Properties, bactypes.Property{
			ID:         property.ObjectList,
			ArrayIndex: uint32(i),
		})
	}
	resp, err := c.ReadMultiProperty(ctx, dev, rpm)
	if err != nil {
		return nil, fmt.Errorf("unable to read multiple properties: %w", err)
	}
	if len(resp.Objects) == 0 {
		return nil, fmt.Errorf("no data was returned")
	}
	objs := make([]bactypes.Object, len(resp.Objects[0].Properties))
	for i, prop := range resp.Objects[0].Properties {
		id, ok := prop.Data.(bactypes.ObjectID)
		if !ok {
			return nil, fmt.Errorf("expected type Object ID, got %T", prop.Data)
		}
		objs[i].ID = id
	}
	return objs, nil
}

func objectCopy(dest bactypes.ObjectMap, src []bactypes.Object) {
	for _, o := range src {
		if dest[o.ID.Type] == nil {
			dest[o.ID.Type] = make(map[bactypes.ObjectInstance]bactypes.Object)
		}
		dest[o.ID.Type][o.ID.Instance] = o
	}
}

func (c *Client) objectList(ctx context.Context, dev *bactypes.Device) error {
	dev.Objects = make(bactypes.ObjectMap)

	l, err := c.objectListLen(ctx, *dev)
	if err != nil {
		return fmt.Errorf("unable to get list length: %w", err)
	}

	scanSize := int(dev.MaxApdu) / readPropRequestSize
	if scanSize < 1 {
		scanSize = 1
	}
	i := 0
	for i = 0; i < l/scanSize; i++ {
		start := i*scanSize + 1
		end := (i + 1) * scanSize
		objs, err := c.objectsRange(ctx, *dev, start, end)
		if err != nil {
			return fmt.Errorf("unable to retrieve objects between %d and %d: %w", start, end, err)
		}
		objectCopy(dev.Objects, objs)
	}
	start := i*scanSize + 1
	end := l
	if start <= end {
		objs, err := c.objectsRange(ctx, *dev, start, end)
		if err != nil {
			return fmt.Errorf("unable to retrieve objects between %d and %d: %w", start, end, err)
		}
		objectCopy(dev.Objects, objs)
	}
	return nil
}

func (c *Client) objectInformation(ctx context.Context, dev *bactypes.Device, objs []bactypes.Object) error {
	keys := make([]bactypes.ObjectID, len(objs))
	counter := 0
	rpm := bactypes.ReadMultipleProperty{Objects: []bactypes.Object{}}
	for _, o := range objs {
		if o.ID.Type > maxStandardBacnetType {
			continue
		}
		keys[counter] = o.ID
		counter++
		rpm.Objects = append(rpm.Objects, bactypes.Object{
			ID: o.ID,
			Properties: []bactypes.Property{
				{ID: property.ObjectName, ArrayIndex: bactypes.ArrayAll},
				{ID: property.Description, ArrayIndex: bactypes.ArrayAll},
			},
		})
	}
	resp, err := c.ReadMultiProperty(ctx, *dev, rpm)
	if err != nil {
		return fmt.Errorf("unable to read multiple property: %w", err)
	}
	for i, r := range resp.Objects {
		name, ok := r.Properties[0].Data.(string)
		if !ok {
			return fmt.Errorf("expecting string got %T", r.Properties[0].Data)
		}
		description, ok := r.Properties[1].Data.(string)
		if !ok {
			return fmt.Errorf("expecting string got %T", r.Properties[1].Data)
		}
		obj := dev.Objects[keys[i].Type][keys[i].Instance]
		obj.Name = name
		obj.Description = description
		dev.Objects[keys[i].Type][keys[i].Instance] = obj
	}
	return nil
}

func (c *Client) allObjectInformation(ctx context.Context, dev *bactypes.Device) error {
	objs := dev.ObjectSlice()
	const incrSize = 5
	for i := 0; i < len(objs); i += incrSize {
		end := i + incrSize
		if end > len(objs) {
			end = len(objs)
		}
		if err := c.objectInformation(ctx, dev, objs[i:end]); err != nil {
			return err
		}
	}
	return nil
}
