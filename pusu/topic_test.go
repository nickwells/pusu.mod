package pusu

import (
	"log/slog"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestTopic(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		topic         Topic
		expectedSlice []Topic
	}{
		{
			ID: testhelper.MkID("empty topic"),
			ExpErr: testhelper.MkExpErr(`bad topic "" - `,
				"it must start with a '/'"),
			topic:         "",
			expectedSlice: []Topic{""},
		},
		{
			ID: testhelper.MkID("non-absolute topic"),
			ExpErr: testhelper.MkExpErr(`bad topic "xxx" - `,
				"it must start with a '/'"),
			topic:         "xxx",
			expectedSlice: []Topic{"xxx"},
		},
		{
			ID:            testhelper.MkID("empty, absolute topic"),
			topic:         "/",
			expectedSlice: []Topic{"/"},
		},
		{
			ID:            testhelper.MkID("absolute topic"),
			topic:         "/a/b/c",
			expectedSlice: []Topic{"/a/b/c", "/a/b", "/a", "/"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.topic.Check()
			testhelper.CheckExpErr(t, err, tc)

			slc := tc.topic.SubTopics()
			testhelper.DiffStringSlice(t,
				tc.IDStr(), "sub-topics",
				slc, tc.expectedSlice)
		})
	}
}

func TestTopicAttr(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		t       Topic
		expAttr slog.Attr
	}{
		{
			ID:      testhelper.MkID("good Topic: /test"),
			t:       Topic("/test"),
			expAttr: slog.String(AttrPfx+"Topic", "/test"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			attr := tc.t.Attr()
			if !attr.Equal(tc.expAttr) {
				t.Log(tc.ID)
				t.Log("\t: expected Attr:", tc.expAttr)
				t.Log("\t:   actual Attr:", attr)
				t.Error("\t: bad attr")
			}
		})
	}
}
