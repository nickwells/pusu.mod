package pusu

import "log/slog"

// NoteTextNamespace provides a narrative description of what a Namespace is.
const NoteTextNamespace = "A publish/subscribe namespace provides a way" +
	" to partition topics in the message broker (the pub/sub server)." +
	"\n\n" +
	"Subscriptions and publications by clients sharing the same namespace" +
	" will relate only to those clients. Two clients having different" +
	" namespaces can each subscribe to the same topic but they will only" +
	" receive messages published on that topic from other clients sharing" +
	" their namespace." +
	"\n\n" +
	"For instance, two clients, c1, in namespace 'A' and" +
	" c2 in namespace 'B' can each subscribe to topic '/T'." +
	" If, subseqently, a third client, also in namespace 'A'" +
	" publishes a message on topic '/T' then only client c1" +
	" will receive the message."

// Namespace represents the namespace to which Topics belong
type Namespace string

// Attr returns an slog Attr representing the Namespace
func (n Namespace) Attr() slog.Attr {
	return slog.String(NamespaceAttrKey, string(n))
}
