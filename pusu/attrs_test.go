package pusu

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestErrorAttr(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		err     error
		expAttr slog.Attr
	}{
		{
			ID:      testhelper.MkID("empty err"),
			expAttr: slog.String("noError", "-"),
		},
		{
			ID:      testhelper.MkID("non-empty err"),
			err:     errors.New("some error"),
			expAttr: slog.String(ErrorAttrKey, "some error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			a := ErrorAttr(tc.err)
			if !a.Equal(tc.expAttr) {
				t.Log(tc.ID)
				t.Log("\t: expected attr:", tc.expAttr)
				t.Log("\t:   actual attr:", a)
				t.Error("\t: attr unexpected")
			}
		})
	}
}
