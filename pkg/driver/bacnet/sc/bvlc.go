package sc

import (
	"encoding/binary"
	"fmt"
)

// BVLC-SC function codes (ASHRAE 135-2020 Clause AB.2.2).
type bvlcFunction uint8

const (
	bvlcResult               bvlcFunction = 0x00
	bvlcEncapsulatedNPDU     bvlcFunction = 0x01
	bvlcAddressResolution    bvlcFunction = 0x02
	bvlcAddressResolutionACK bvlcFunction = 0x03
	bvlcAdvertisement        bvlcFunction = 0x04
	bvlcAdvertisementSolicit bvlcFunction = 0x05
	bvlcConnectRequest       bvlcFunction = 0x06
	bvlcConnectAccept        bvlcFunction = 0x07
	bvlcDisconnectRequest    bvlcFunction = 0x08
	bvlcDisconnectACK        bvlcFunction = 0x09
	bvlcHeartbeatRequest     bvlcFunction = 0x0A
	bvlcHeartbeatACK         bvlcFunction = 0x0B
	bvlcProprietaryMessage   bvlcFunction = 0x0C
)

func (f bvlcFunction) String() string {
	switch f {
	case bvlcResult:
		return "Result"
	case bvlcEncapsulatedNPDU:
		return "Encapsulated-NPDU"
	case bvlcAddressResolution:
		return "Address-Resolution"
	case bvlcAddressResolutionACK:
		return "Address-Resolution-ACK"
	case bvlcAdvertisement:
		return "Advertisement"
	case bvlcAdvertisementSolicit:
		return "Advertisement-Solicitation"
	case bvlcConnectRequest:
		return "Connect-Request"
	case bvlcConnectAccept:
		return "Connect-Accept"
	case bvlcDisconnectRequest:
		return "Disconnect-Request"
	case bvlcDisconnectACK:
		return "Disconnect-ACK"
	case bvlcHeartbeatRequest:
		return "Heartbeat-Request"
	case bvlcHeartbeatACK:
		return "Heartbeat-ACK"
	case bvlcProprietaryMessage:
		return "Proprietary-Message"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", uint8(f))
	}
}

// Control octet flags (Clause AB.2.1.1.1). Bits 4-7 are reserved.
const (
	ctrlDataOptions = 0x01 // data options present
	ctrlDestOptions = 0x02 // destination-specific options present
	ctrlDestVAddr   = 0x04 // destination virtual address present
	ctrlOrigVAddr   = 0x08 // originating virtual address present
)

// bvlcMessage is a decoded BACnet/SC BVLC message. Header options are not used
// by this driver and are rejected on decode if present.
type bvlcMessage struct {
	Function  bvlcFunction
	MessageID uint16
	HasOrig   bool
	Orig      VMAC // originating (source) virtual address, valid when HasOrig
	HasDest   bool
	Dest      VMAC // destination virtual address, valid when HasDest
	Payload   []byte
}

// encode serialises the message to wire format.
func (m bvlcMessage) encode() []byte {
	var control byte
	if m.HasOrig {
		control |= ctrlOrigVAddr
	}
	if m.HasDest {
		control |= ctrlDestVAddr
	}
	buf := make([]byte, 0, 4+12+len(m.Payload))
	buf = append(buf, byte(m.Function), control)
	buf = binary.BigEndian.AppendUint16(buf, m.MessageID)
	if m.HasOrig {
		buf = append(buf, m.Orig[:]...)
	}
	if m.HasDest {
		buf = append(buf, m.Dest[:]...)
	}
	buf = append(buf, m.Payload...)
	return buf
}

// decodeBVLC parses a BACnet/SC BVLC message from b.
func decodeBVLC(b []byte) (bvlcMessage, error) {
	var m bvlcMessage
	if len(b) < 4 {
		return m, fmt.Errorf("bvlc message too short: %d octets", len(b))
	}
	m.Function = bvlcFunction(b[0])
	control := b[1]
	m.MessageID = binary.BigEndian.Uint16(b[2:4])
	off := 4
	if control&ctrlOrigVAddr != 0 {
		if len(b) < off+6 {
			return m, fmt.Errorf("bvlc %s: truncated originating vmac", m.Function)
		}
		m.HasOrig = true
		copy(m.Orig[:], b[off:off+6])
		off += 6
	}
	if control&ctrlDestVAddr != 0 {
		if len(b) < off+6 {
			return m, fmt.Errorf("bvlc %s: truncated destination vmac", m.Function)
		}
		m.HasDest = true
		copy(m.Dest[:], b[off:off+6])
		off += 6
	}
	if control&(ctrlDestOptions|ctrlDataOptions) != 0 {
		// Header options are not produced or consumed by this driver. Skipping
		// them safely requires fully parsing the option list, so reject instead
		// of risking misalignment.
		return m, fmt.Errorf("bvlc %s: header options are not supported", m.Function)
	}
	m.Payload = b[off:]
	return m, nil
}

// connectInfo is the payload of Connect-Request and Connect-Accept
// (Clauses AB.2.11 / AB.2.12).
type connectInfo struct {
	VMAC          VMAC
	DeviceUUID    [16]byte
	MaxBVLCLength uint16
	MaxNPDULength uint16
}

func (c connectInfo) encode() []byte {
	buf := make([]byte, 0, 6+16+2+2)
	buf = append(buf, c.VMAC[:]...)
	buf = append(buf, c.DeviceUUID[:]...)
	buf = binary.BigEndian.AppendUint16(buf, c.MaxBVLCLength)
	buf = binary.BigEndian.AppendUint16(buf, c.MaxNPDULength)
	return buf
}

func decodeConnectInfo(b []byte) (connectInfo, error) {
	var c connectInfo
	if len(b) < 6+16+2+2 {
		return c, fmt.Errorf("connect payload too short: %d octets", len(b))
	}
	copy(c.VMAC[:], b[0:6])
	copy(c.DeviceUUID[:], b[6:22])
	c.MaxBVLCLength = binary.BigEndian.Uint16(b[22:24])
	c.MaxNPDULength = binary.BigEndian.Uint16(b[24:26])
	return c, nil
}

// resultInfo is the BVLC-Result payload (Clause AB.2.4). Only the NAK fields we
// surface in errors are parsed.
type resultInfo struct {
	Function   bvlcFunction // the function this result responds to
	Result     uint8        // 0 = ACK, 1 = NAK
	ErrorClass uint16
	ErrorCode  uint16
}

func decodeResult(b []byte) (resultInfo, error) {
	var r resultInfo
	if len(b) < 2 {
		return r, fmt.Errorf("result payload too short: %d octets", len(b))
	}
	r.Function = bvlcFunction(b[0])
	r.Result = b[1]
	if r.Result != 0 && len(b) >= 7 {
		// b[2] is the error header marker, then error class + code (2 octets each).
		r.ErrorClass = binary.BigEndian.Uint16(b[3:5])
		r.ErrorCode = binary.BigEndian.Uint16(b[5:7])
	}
	return r, nil
}
