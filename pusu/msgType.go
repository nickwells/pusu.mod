package pusu

import (
	"fmt"
	"log/slog"
)

// MsgType represents the type of the client message
type MsgType uint8

const (
	// MinMsgType should always be the first entry in this list and is used
	// to verify that the message is well formed - it is not a valid message
	// type and all message types must be greater than this value
	MinMsgType MsgType = iota
	// Identify is the first message received which identifies the connection
	Identify
	// Publish represents a publication
	Publish
	// Subscribe represents a collection of subscription requests
	Subscribe
	// Unsubscribe is sent to cancel a subscription
	Unsubscribe
	// UnsubscribeAll is sent to cancel all a client's subscriptions
	UnsubscribeAll
	// Ping is a test-of-life/proof-of-life  message
	Ping
	// MaxMsgType should always be the last entry in this list and is used to
	// verify that the message is well formed - it is not a valid message
	// type and all message types must be less than this value
	MaxMsgType
)

// Check will return a non-nil error if the message type is invalid
func (mt MsgType) Check() error {
	if mt <= MinMsgType {
		return fmt.Errorf("bad message type: %d - too small", mt)
	}

	if mt >= MaxMsgType {
		return fmt.Errorf("bad message type: %d - too large", mt)
	}

	return nil
}

// Attr returns a slog Attr representing a message type
func (mt MsgType) Attr() slog.Attr {
	return slog.String(PuSuPrefix+"MsgType", fmt.Sprintf("%d(%s)", mt, mt))
}
