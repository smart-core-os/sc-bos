package opcua

import (
	"context"
	"errors"
	"slices"

	"github.com/gopcua/opcua/ua"
)

// severityMask isolates the severity bits (31:30) of an OPC UA StatusCode.
// See OPC UA Part 4 s7.34: the low bits carry sub-codes and info bits that
// do not affect whether the value is usable. A value delivered through a
// subscription commonly arrives as Good with the Overflow info bit set (0x480),
// which is still a perfectly good value.
const severityMask ua.StatusCode = 0xC0000000

// statusIsGood reports whether the status says the value is usable as-is.
func statusIsGood(c ua.StatusCode) bool { return c&severityMask == ua.StatusGood }

// statusIsUncertain reports whether the status says the value is usable but of reduced quality.
func statusIsUncertain(c ua.StatusCode) bool { return c&severityMask == ua.StatusUncertain }

// statusIsBad reports whether the status says the value should not be used.
func statusIsBad(c ua.StatusCode) bool { return c&severityMask == ua.StatusBad }

// statusCodeMask isolates the code part of a StatusCode, dropping the sub-code and info bits.
// Every ua.Status* constant is of the form 0xNNNN0000, and a server is free to attach info
// bits to the code it sends (Part 4 s7.34), so a code has to be masked before it is compared
// against the constants.
const statusCodeMask ua.StatusCode = 0xFFFF0000

// permanentSubscribeStatuses are the status codes that say a subscription to a point can never
// succeed, however long we wait: the node does not exist, the attribute or encoding we asked
// for is wrong, the server does not implement what we need, or we are not allowed to read it.
// Retrying any of these just asks the same question again forever.
//
// This is deliberately a deny-list, with everything unrecognised treated as retryable, because
// the two mistakes are not symmetric. A retryable code missing from a hypothetical allow-list
// would silently abandon a working point for the lifetime of the config, which is the bug this
// retry loop exists to fix. A permanent code missing from this list costs one request per
// backoff period and a fault naming the point, which is loud and cheap.
//
// Retryable by design, and worth naming because they are the ones seen in the field:
// StatusBadTimeout, which is usually our own client-side RequestTimeout expiring against a
// slow server rather than anything the server said; the capacity family
// (BadTooManyOperations, BadTooManyMonitoredItems, BadTooManySubscriptions,
// BadResourceUnavailable, BadOutOfMemory), which clears as load drops; the connection and
// session family, which gopcua's AutoReconnect is already repairing underneath us; and
// BadNoCommunication and BadWaitingForInitialData, which are routine on a DA-wrapper server
// that has not finished polling its underlying devices yet.
var permanentSubscribeStatuses = []ua.StatusCode{
	ua.StatusBadNodeIDInvalid, ua.StatusBadNodeIDUnknown, ua.StatusBadAttributeIDInvalid,
	ua.StatusBadIndexRangeInvalid, ua.StatusBadDataEncodingInvalid, ua.StatusBadDataEncodingUnsupported,
	ua.StatusBadNotReadable, ua.StatusBadNotSupported, ua.StatusBadNotImplemented, ua.StatusBadTypeMismatch,
	ua.StatusBadMonitoringModeInvalid, ua.StatusBadMonitoredItemFilterUnsupported,
	ua.StatusBadFilterNotAllowed, ua.StatusBadNothingToDo, ua.StatusBadServiceUnsupported,
	ua.StatusBadUserAccessDenied, ua.StatusBadIdentityTokenInvalid, ua.StatusBadIdentityTokenRejected,
}

var (
	// errUnexpectedResults reports a CreateMonitoredItems response that did not carry one
	// result per item we asked about, so we cannot tell which of them worked.
	errUnexpectedResults = errors.New("expected one monitored item result per item requested")
	// errSubscriptionClosed reports the notification channel closing under us.
	errSubscriptionClosed = errors.New("subscription notification channel closed")
	// errNoWorkablePoints reports a device left with nothing to subscribe to, because the
	// server has said no to every one of its points for a reason that will never change.
	errNoWorkablePoints = errors.New("no point on this device can be monitored")
)

// subscribeErrIsPermanent reports whether err says subscribing to a point can never succeed,
// so there is no point retrying it.
//
// Only a status code the server sent can say that, so anything else - a nil error, a cancelled
// context, a transport failure - reports false and is retried. ua.StatusCode has a value
// receiver Error method, so it satisfies error and errors.As finds it through the %w wrapping
// Client.Subscribe applies.
func subscribeErrIsPermanent(err error) bool {
	if err == nil {
		return false
	}
	// a context ending is us stopping, not the server refusing
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var status ua.StatusCode
	if !errors.As(err, &status) {
		return false
	}
	return slices.Contains(permanentSubscribeStatuses, status&statusCodeMask)
}
