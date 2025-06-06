package pusu

import "log/slog"

// Namespace represents the namespace to which Topics belong
type Namespace string

// Attr returns an slog Attr representing the Namespace
func (n Namespace) Attr() slog.Attr {
	return slog.String(NamespaceAttrKey, string(n))
}
