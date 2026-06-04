package main

// Write support for bacnet-multi-comm-test.
//
// A `-writes-file` flag points to a JSON file describing WriteProperty
// operations to execute against the discovered devices. Each entry targets a
// device by its BACnet instance id (must match a device in one of the loaded
// configs); writes are executed after the read pass for that device, on the
// same client, and the per-write outcome is appended to a parallel
// writes_results CSV.

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/smart-core-os/gobacnet/property"
	bactypes "github.com/smart-core-os/gobacnet/types"

	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/bclient"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
)

// WriteOp describes a single WriteProperty operation loaded from -writes-file.
type WriteOp struct {
	DeviceID  uint32      `json:"deviceId"`
	ObjectID  string      `json:"objectId"`            // e.g. "AnalogValue:1"
	Property  string      `json:"property,omitempty"`  // name or numeric id; defaults to present-value
	Value     interface{} `json:"value"`               // JSON-decoded; coerced per ValueType
	ValueType string      `json:"valueType,omitempty"` // real|unsigned|signed|boolean|string|null (default real)
	Priority  uint        `json:"priority,omitempty"`  // 1..16; defaults to 16
}

// writeResult is the outcome of executing one WriteOp.
type writeResult struct {
	Op  WriteOp
	OK  bool
	Err string
}

// loadWrites reads and validates the writes file. Returns nil writes when path is empty.
func loadWrites(path string) ([]WriteOp, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ops []WriteOp
	if err := json.Unmarshal(data, &ops); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for i := range ops {
		if ops[i].DeviceID == 0 {
			return nil, fmt.Errorf("writes[%d]: deviceId is required", i)
		}
		if ops[i].ObjectID == "" {
			return nil, fmt.Errorf("writes[%d]: objectId is required", i)
		}
		if ops[i].Property == "" {
			ops[i].Property = "present-value"
		}
		if ops[i].ValueType == "" {
			ops[i].ValueType = "real"
		}
		if ops[i].Priority == 0 {
			ops[i].Priority = 16
		}
	}
	return ops, nil
}

// writesByDevice indexes writes by device instance id.
func writesByDevice(ops []WriteOp) map[uint32][]WriteOp {
	out := make(map[uint32][]WriteOp, len(ops))
	for _, op := range ops {
		out[op.DeviceID] = append(out[op.DeviceID], op)
	}
	return out
}

// executeWrites runs every write targeted at dev and returns one writeResult per op.
func executeWrites(ctx context.Context, client bclient.Client, dev bactypes.Device, ops []WriteOp) []writeResult {
	results := make([]writeResult, 0, len(ops))
	for _, op := range ops {
		results = append(results, executeWrite(ctx, client, dev, op))
	}
	return results
}

func executeWrite(ctx context.Context, client bclient.Client, dev bactypes.Device, op WriteOp) writeResult {
	objID, err := config.ObjectIDFromString(op.ObjectID)
	if err != nil {
		return failed(op, fmt.Errorf("objectId: %w", err))
	}
	propID, err := parsePropertyID(op.Property)
	if err != nil {
		return failed(op, fmt.Errorf("property: %w", err))
	}
	value, err := coerceValue(op.ValueType, op.Value)
	if err != nil {
		return failed(op, fmt.Errorf("value/valueType: %w", err))
	}

	wp := bactypes.ReadPropertyData{
		Object: bactypes.Object{
			ID: bactypes.ObjectID(objID),
			Properties: []bactypes.Property{
				{ID: propID, ArrayIndex: bactypes.ArrayAll, Data: value},
			},
		},
	}
	wctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	if err := client.WriteProperty(wctx, dev, wp, op.Priority); err != nil {
		log.Printf("WRITE device=%d obj=%s prop=%s -> ERR: %v", op.DeviceID, op.ObjectID, op.Property, err)
		return failed(op, err)
	}
	log.Printf("WRITE device=%d obj=%s prop=%s value=%v prio=%d -> OK", op.DeviceID, op.ObjectID, op.Property, op.Value, op.Priority)
	return writeResult{Op: op, OK: true}
}

func failed(op WriteOp, err error) writeResult {
	return writeResult{Op: op, OK: false, Err: err.Error()}
}

func parsePropertyID(s string) (property.ID, error) {
	if id, ok := property.FromString(s); ok {
		return id, nil
	}
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("unknown property name or id %q", s)
	}
	return property.ID(n), nil
}

// coerceValue converts a JSON-decoded value into the concrete Go type the
// gobacnet encoder maps to the requested BACnet primitive.
func coerceValue(typ string, raw interface{}) (interface{}, error) {
	switch typ {
	case "real":
		f, err := asFloat(raw)
		if err != nil {
			return nil, err
		}
		return float32(f), nil
	case "unsigned":
		f, err := asFloat(raw)
		if err != nil {
			return nil, err
		}
		if f < 0 {
			return nil, fmt.Errorf("negative value for unsigned: %v", raw)
		}
		return uint32(f), nil
	case "signed":
		f, err := asFloat(raw)
		if err != nil {
			return nil, err
		}
		return int32(f), nil
	case "boolean":
		switch v := raw.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		default:
			return nil, fmt.Errorf("expected boolean, got %T", raw)
		}
	case "string":
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", raw)
		}
		return s, nil
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported valueType %q", typ)
	}
}

func asFloat(raw interface{}) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("not numeric: %T", raw)
	}
}

// writeWriteResults writes the per-op outcomes to a parallel CSV next to the
// reads results file (resultsFile-stem + "_writes_" + timestamp + ".csv").
func writeWriteResults(fileName string, results []writeResult) error {
	if len(results) == 0 {
		return nil
	}
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"DeviceID", "ObjectID", "Property", "Value", "ValueType", "Priority", "OK", "Error"}); err != nil {
		return err
	}
	for _, r := range results {
		if err := w.Write([]string{
			fmt.Sprintf("%d", r.Op.DeviceID),
			r.Op.ObjectID,
			r.Op.Property,
			fmt.Sprintf("%v", r.Op.Value),
			r.Op.ValueType,
			fmt.Sprintf("%d", r.Op.Priority),
			boolYesNo(r.OK),
			r.Err,
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
