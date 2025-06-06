package pusu

import (
	"fmt"
	"log/slog"
)

// MsgType represents the type of the client message
type MsgType uint8

const (
	// Invalid should always be the first entry in this list and is used
	// to verify that the message is well formed - it is not a valid message
	// type and all message types must be greater than this value.
	Invalid MsgType = iota
	// Start is the first message sent by the client. It identifies the
	// connection and registers the namespace in which topics are registered.
	Start
	// Publish represents a publication. The message will be sent to all the
	// clients who have subscribed to the publication topic.
	Publish
	// Subscribe represents a collection of subscriptions to topics.
	Subscribe
	// Unsubscribe is sent to cancel a collection of subscriptions
	Unsubscribe
	// Ping is a test-of-life/proof-of-life message
	Ping
	// Error is a message from the server indicating that some error has
	// occurred. The client will disconnect if this message is received.
	Error
	// Ack is a message from the server to acknowledge that a message has
	// been received and processed. Every message (except Pings) sent to the
	// server will result in either an Ack or an Error.
	Ack
	// MaxMsgType should always be the last entry in this list and is used to
	// verify that the message is well formed - it is not a valid message
	// type and all message types must be less than this value
	MaxMsgType
)

// Check will return a non-nil error if the message type is invalid
func (mt MsgType) Check() error {
	// MsgType is unsigned and Invalid is the zero value so there is no need
	// to check for "less than"
	if mt == Invalid {
		return fmt.Errorf("bad message type: %s", mt)
	}

	if mt >= MaxMsgType {
		return fmt.Errorf("bad message type: %s - too big", mt)
	}

	return nil
}

// Attr returns a slog Attr representing a message type
func (mt MsgType) Attr() slog.Attr {
	return slog.String(AttrPfx+"MsgType", fmt.Sprintf("%d(%s)", mt, mt))
}
