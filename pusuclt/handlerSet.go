package pusuclt

import (
	"errors"
)

var (
	errHandlerAlreadyAdded = errors.New("the handler has already been added")
	errHandlerNotInSet     = errors.New("the handler is not in the handler set")
)

// handlerIndexes maps between a MsgHandler address and the index into the
// handlersInOrder slice in the handlerSet struct
type handlerIndexes map[uintptr]int

// handlerSet represents the collection of handler funcs for messages
// received on a Topic. This allows us to identify whether or not the
// handler has been provided already so we can have multiple handlers for the
// same topic.
type handlerSet struct {
	// handlersInOrder gives the message handler functions in their
	// subcription order. Unsubscribing will set this entry to nil. If the
	// entry marked as nil is the last entry in this slice it is deleted from
	// the slice and the resulting last entry is checked to see if it is nil
	// and if so that is deleted too and so on. When the slice is empty the
	// Unsubscribe message is sent to the pub/sub server
	handlersInOrder []MsgHandler
	// handlerMap gives the index in the slice for the MsgHandler
	// . Unsubscribing will use this entry to find the slice entry to set to
	// nil and then the map entry will be deleted
	handlerMap handlerIndexes
}

// newHandlerSet returns a properly instantiated handlerSet
func newHandlerSet() *handlerSet {
	return &handlerSet{
		handlersInOrder: []MsgHandler{},
		handlerMap:      make(handlerIndexes),
	}
}

// handlerCount returns the number of handlers
func (hs handlerSet) handlerCount() int {
	return len(hs.handlerMap)
}

// removeTrailingNils removes all the entries at the end of handlersInOrder
// which are set to nil. This is to remove unnecessary checking of nil
// MsgHamdlers when processing a received message. Nil handlers not at the
// end cannot be removed without a potentially expensive re-indexing of all
// the existing handlers in the handlerMap.
func (hs *handlerSet) removeTrailingNils() {
	count := 0

	for i := len(hs.handlersInOrder) - 1; i >= 0; i-- {
		if hs.handlersInOrder[i] != nil {
			break
		}

		count++
	}

	if count == 0 {
		return
	}

	hs.handlersInOrder = hs.handlersInOrder[:len(hs.handlersInOrder)-count]
}

// addHandler adds the handler to the handlerSet. It returns a non-nil error
// if the handler is already in the handler map.
func (hs *handlerSet) addHandler(h MsgHandler) error {
	hID := h.id()

	if _, ok := hs.handlerMap[hID]; ok {
		return errHandlerAlreadyAdded
	}

	hs.handlerMap[hID] = len(hs.handlersInOrder)
	hs.handlersInOrder = append(hs.handlersInOrder, h)

	return nil
}

// removeHandler removes the identified handler from the handlerSet. It
// returns a non-nil error if the handler is not found in the handler map.
func (hs *handlerSet) removeHandler(h MsgHandler) error {
	hID := h.id()

	hIdx, ok := hs.handlerMap[hID]
	if !ok {
		return errHandlerNotInSet
	}

	delete(hs.handlerMap, hID)
	hs.handlersInOrder[hIdx] = nil
	hs.removeTrailingNils()

	return nil
}
