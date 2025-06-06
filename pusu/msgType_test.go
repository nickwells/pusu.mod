package pusu

import (
	"log/slog"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestMsgTypeCheck(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		mt MsgType
	}{
		{
			ID: testhelper.MkID("good - Start"),
			mt: Start,
		},
		{
			ID: testhelper.MkID("good - Error"),
			mt: Error,
		},
		{
			ID:     testhelper.MkID("bad - Invalid"),
			ExpErr: testhelper.MkExpErr("bad message type: Invalid"),
			mt:     Invalid,
		},
		{
			ID:     testhelper.MkID("bad - MaxMsgType"),
			ExpErr: testhelper.MkExpErr("bad message type: MaxMsgType"),
			mt:     MaxMsgType,
		},
		{
			ID:     testhelper.MkID("bad - MaxMsgType+1"),
			ExpErr: testhelper.MkExpErr("bad message type: MsgType(99)"),
			mt:     MsgType(99),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.mt.Check()
			testhelper.CheckExpErr(t, err, tc)
		})
	}
}

func TestMsgTypeAttr(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		mt      MsgType
		expAttr slog.Attr
	}{
		{
			ID:      testhelper.MkID("good MsgType: Start"),
			mt:      Start,
			expAttr: slog.String(AttrPfx+"MsgType", "1(Start)"),
		},
		{
			ID:      testhelper.MkID("bad MsgType: Invalid"),
			mt:      Invalid,
			expAttr: slog.String(AttrPfx+"MsgType", "0(Invalid)"),
		},
		{
			ID:      testhelper.MkID("bad MsgType: 99"),
			mt:      MsgType(99),
			expAttr: slog.String(AttrPfx+"MsgType", "99(MsgType(99))"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			attr := tc.mt.Attr()
			if !attr.Equal(tc.expAttr) {
				t.Log(tc.ID)
				t.Log("\t: expected Attr:", tc.expAttr)
				t.Log("\t:   actual Attr:", attr)
				t.Error("\t: bad attr")
			}
		})
	}
}
