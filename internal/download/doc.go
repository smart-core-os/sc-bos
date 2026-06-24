// Package download serves file downloads by generating signed capability
// URLs encoding requests of what to download.
//
// The package is intended for HTTP endpoints where:
//
//   - A gRPC (or similar) API decides whether a client may download
//     something and embeds the parameters of that download in a signed,
//     time-limited token; and
//   - An unauthenticated HTTP endpoint serves the file from that token
//     alone, with no further authorisation check.
//
// # Wire format
//
// A download URL ends with a single token of the form:
//
//	<b64url(envelope)>.<b64url(signature)>
//
// where envelope is a marshalled Envelope message (type + payload + expiry)
// and signature is computed by the configured Signer over the raw envelope
// bytes. The signature is verified before the envelope is decoded.
//
// # Type strings
//
// The type string passed to Handle is embedded verbatim in every URL the
// router signs. Use stable identifiers chosen at the consumer
// (e.g. "devices-csv", "system-log", "audit-log") — they are part of the wire
// format and changing one invalidates outstanding URLs.
package download
