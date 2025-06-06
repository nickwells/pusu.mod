package pusuclt

import (
	"strings"
	"testing"
)

func TestClientID(t *testing.T) {
	prefixes := []string{
		clientIDProgramPfx,
		clientIDHostPfx,
		clientIDUserPfx,
		clientIDPidPfx,
	}
	cid := makeClientID("test-progname")
	cidParts := strings.Split(cid, clientIDSep)

	if len(cidParts) != len(prefixes) {
		t.Log("the number of parts in the clientID is wrong")
		t.Log("\t:", cid)
		t.Logf("\t: expected %d\n", len(prefixes))
		t.Logf("\t:   actual %d\n", len(cidParts))
		t.Error("\t: failed to create clientID")
	} else {
		for i, part := range cidParts {
			expPfx := prefixes[i]
			if !strings.HasPrefix(part, expPfx) {
				t.Logf("the prefix for part %d of the clientID is wrong", i)
				t.Logf("\t: expected %s\n", expPfx)
				t.Logf("\t:   actual %s\n", part)
				t.Error("\t: failed to create clientID")
			}
		}
	}
}
