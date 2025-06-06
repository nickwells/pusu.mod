package pusuclt

import (
	"testing"

	"github.com/nickwells/pusu.mod/pusu"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestTopicHandlerCheck(t *testing.T) {
	handler := func(_ pusu.Topic, _ []byte) {}

	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		th TopicHandler
	}{
		{
			ID: testhelper.MkID("bad topic"),
			ExpErr: testhelper.MkExpErr(
				`bad topic "bad" - it must start with a '/'`),
			th: TopicHandler{
				Topic:   "bad",
				Handler: handler,
			},
		},
		{
			ID: testhelper.MkID("bad handler"),
			ExpErr: testhelper.MkExpErr(
				`the MsgHandler for Topic "/good" is nil`),
			th: TopicHandler{
				Topic:   "/good",
				Handler: nil,
			},
		},
		{
			ID: testhelper.MkID("good TopicHandler"),
			th: TopicHandler{
				Topic:   "/good",
				Handler: handler,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.th.check()
			testhelper.CheckExpErr(t, err, tc)
		})
	}
}

func TestTopicHandlerID(t *testing.T) {
	handler := func(_ pusu.Topic, _ []byte) {}

	testCases := []struct {
		testhelper.ID
		th         TopicHandler
		expectZero bool
	}{
		{
			ID: testhelper.MkID("good handler"),
			th: TopicHandler{
				Topic:   "/good",
				Handler: handler,
			},
			expectZero: false,
		},
		{
			ID: testhelper.MkID("bad handler"),
			th: TopicHandler{
				Topic:   "/good",
				Handler: nil,
			},
			expectZero: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			id := tc.th.id()
			if id == 0 && !tc.expectZero {
				t.Log(tc.ID)
				t.Error("\t: the id is zero, it should not be")
			} else if id != 0 && tc.expectZero {
				t.Log(tc.ID)
				t.Error("\t: the id is not zero, it should be")
			}
		})
	}
}
