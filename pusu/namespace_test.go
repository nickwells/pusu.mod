package pusu

import (
	"log/slog"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestNamespaceAttr(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		n       Namespace
		expAttr slog.Attr
	}{
		{
			ID:      testhelper.MkID("good Namespace: test"),
			n:       Namespace("test"),
			expAttr: slog.String(AttrPfx+"Namespace", "test"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			attr := tc.n.Attr()
			if !attr.Equal(tc.expAttr) {
				t.Log(tc.ID)
				t.Log("\t: expected Attr:", tc.expAttr)
				t.Log("\t:   actual Attr:", attr)
				t.Error("\t: bad attr")
			}
		})
	}
}
