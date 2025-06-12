package pusu

import (
	"fmt"
	"log/slog"
	"path"
)

// NoteTextTopic provides a narrative description of what a Topic is.
const NoteTextTopic = "A publish/subscribe topic provides routing information" +
	" for the message broker (the pub/sub server) to decide where to send" +
	" messages it has received." +
	"\n\n" +
	"A client can subscribe to a topic and then any messages which are" +
	" subsequently published on that topic will be sent to that client." +
	" To stop receiving such messages the client will need to" +
	" unsubsubscribe from that topic." +
	"\n\n" +
	"A topic name must be a well-formed path, starting with a '/' and" +
	" having one or more parts following this, each part separated from its" +
	" predecessor by a single '/'."

// Topic represents a pub/sub topic. Clients subscribe to topics and publish
// messages on the topics and the pub/sub server distributes those messages
// to any clients subscribed to the topic. Note that a valid topic must be a
// 'Clean', 'Absolute' path, for instance '/a/b/c' is valid but '//a' is not,
// nor is 'a/b'; see the [path] package for details.
type Topic string

// Attr creates a standard slog Attr representing the topic
func (t Topic) Attr() slog.Attr {
	return slog.String(AttrPfx+"Topic", string(t))
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

// SubTopics progressively strips the last part of the topic path and returns
// a slice of the resultant topics. So if you pass it a topic '/a/b/c' it
// will return a slice: [ '/a/b/c', '/a/b', '/a', '/' ].
func (t Topic) SubTopics() []Topic {
	subTopics := []Topic{t}

	if err := t.Check(); err != nil {
		return subTopics
	}

TopicLoop:
	for {
		newTopic := Topic(path.Dir(string(t)))

		if newTopic == t {
			break TopicLoop
		}

		t = newTopic
		subTopics = append(subTopics, t)
	}

	return subTopics
}

// stdErr returns a standardised error representing a problem with the topic
func (t Topic) stdErr(text string) error {
	return fmt.Errorf("bad topic %q - %s", t, text)
}
