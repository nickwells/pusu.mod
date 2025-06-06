package pusuclt

import (
	"fmt"

	"github.com/nickwells/pusu.mod/pusu"
)

// TopicHandler represents an association between a topic and a handler for
// the messages expected to be received over that topic.
type TopicHandler struct {
	Topic   pusu.Topic
	Handler MsgHandler
}

// check tests the TopicHandler for validity - the Topic must pass its checks
// and the MsgHandler must not be nil
func (th TopicHandler) check() error {
	if err := th.Topic.Check(); err != nil {
		return err
	}

	if th.Handler == nil {
		return fmt.Errorf("the MsgHandler for Topic %q is nil", th.Topic)
	}

	return nil
}

// id returns the id of the MsgHandler
func (th TopicHandler) id() uintptr {
	return th.Handler.id()
}

// String returns a string representation of the TopicHandler
func (th TopicHandler) String() string {
	if th.Handler == nil {
		return fmt.Sprintf("TopicHandler{Topic: %q, Handler: nil}", th.Topic)
	}

	return fmt.Sprintf("TopicHandler{Topic: %q, Handler: 0x%x}",
		th.Topic, th.id())
}
