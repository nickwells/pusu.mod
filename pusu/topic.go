package pusu

import (
	"fmt"
	"log/slog"
	"path"
)

// Topic represents a pub/sub topic. Clients subscribe to topics and publish
// messages on the topics and the pub/sub server distributes those messages
// to any clients subscribed to the topic. Note that a valid topic must be a
// 'Clean', 'Absolute' path, for instance '/a/b/c' is valid but '//a' is not,
// nor is 'a/b'; see the [path] package for details.
type Topic string

// Attr creates a standard slog Attr representing the topic
func (t Topic) Attr() slog.Attr {
	return slog.String(PuSuPrefix+"Topic", string(t))
}

// Check returns a non-nil error if the topic is invalid
func (t Topic) Check() error {
	if !path.IsAbs(string(t)) {
		return t.stdErr("it must start with a '/'")
	}

	tStr := string(t)
	cleanTopic := path.Clean(tStr)
	if tStr != cleanTopic {
		return t.stdErr(fmt.Sprintf("unclean - replace with %q", cleanTopic))
	}

	return nil
}

// stdErr returns a standardised error representing a problem with the topic
func (t Topic) stdErr(text string) error {
	return fmt.Errorf("bad topic %q - %s", t, text)
}
