package pusu

import "log/slog"

// PusuPrefix is a standard prefix to apply to Attr Key names where the Attr
// is pub/sub related
const PuSuPrefix = "pusu."

// ErrorAttr creates a standard slog Attr from the error. It is slightly more
// concise and the Attr key name is standardised. If the err value is nil a
// 'no-error' Attr is returned.
func ErrorAttr(err error) slog.Attr {
	if err != nil {
		return slog.String("error", err.Error())
	}

	return slog.String("no-error", "-")
}

// PemFileAttr creates a standard slog Attr for a PEM file name
func PemFileAttr(filename string) slog.Attr {
	return slog.String("PEM-filename", filename)
}

// NetAddressAttr creates a standard slog Attr for a PEM file name
func NetAddressAttr(addr string) slog.Attr {
	return slog.String("net-address", addr)
}
