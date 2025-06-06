package pusu

import "log/slog"

const (
	// AttrPfx is a standard prefix to apply to Attr Key names where
	// the Attr is pub/sub related
	AttrPfx = "PubSub-"

	// NetAddressAttrKey is the key used for the network address slog.Attr
	NetAddressAttrKey = AttrPfx + "Net-Address"

	// NamespaceAttrKey is the key used for the namespace slog.Attr
	NamespaceAttrKey = AttrPfx + "Namespace"

	// ErrorAttrKey is the key used for a slog.Attr used for reporting an
	// error
	ErrorAttrKey = "error"

	// NoErrorAttrKey is the key used for a slog.Attr used when an ErrorAttr
	// was created but the error given was nil
	NoErrorAttrKey = "noError"
)

// ErrorAttr creates a standard slog Attr from the error. It is slightly more
// concise and the Attr key name is standardised. If the err value is nil an
// Attr with key name 'no-error' is returned.
func ErrorAttr(err error) slog.Attr {
	if err != nil {
		return slog.String("error", err.Error())
	}

	return slog.String("noError", "-")
}

// PemFileAttr creates a standard slog Attr for a PEM file name
func PemFileAttr(filename string) slog.Attr {
	return slog.String("PEM-filename", filename)
}

// NetAddressAttr creates a standard slog Attr for a net address
func NetAddressAttr(addr string) slog.Attr {
	return slog.String(NetAddressAttrKey, addr)
}
