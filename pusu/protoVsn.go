package pusu

import "log/slog"

// CurrentProtoVsn is the version of the protocol implemented by this
// package. It is passed in the Start message to let the server know what
// protocol to expect. A server may choose to support more than the latest
// protocol version.
const CurrentProtoVsn = 1

// The ProtoVsn records the version of the pub/sub protocol being used
type ProtoVsn int32

// Attr returns a slog.Attr describing the protocol version
func (pv ProtoVsn) Attr() slog.Attr {
	return slog.Int(AttrPfx+"ProtoVsn", int(pv))
}
