package pusuclt

import (
	"testing"

	"github.com/nickwells/pusu.mod/pusu"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestAllHandlers(t *testing.T) {
	mh1 := func(_ pusu.Topic, _ []byte) {}
	mh2 := func(_ pusu.Topic, _ []byte) {}
	mh3 := func(_ pusu.Topic, _ []byte) {}

	const (
		add = iota
		remove
	)

	type addRemove struct {
		addOrRemove int
		mh          MsgHandler
	}

	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		work                  []addRemove
		expCount              int
		expHandlersInOrderLen int
	}{
		{
			ID: testhelper.MkID("one new handler"),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
			},
			expCount:              1,
			expHandlersInOrderLen: 1,
		},
		{
			ID: testhelper.MkID("add handler, remove it and add it again"),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: remove,
					mh:          mh1,
				},
				{
					addOrRemove: add,
					mh:          mh1,
				},
			},
			expCount:              1,
			expHandlersInOrderLen: 1,
		},
		{
			ID: testhelper.MkID("two new handlers"),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: add,
					mh:          mh2,
				},
			},
			expCount:              2,
			expHandlersInOrderLen: 2,
		},
		{
			ID: testhelper.MkID("two new handlers, remove first"),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: add,
					mh:          mh2,
				},
				{
					addOrRemove: remove,
					mh:          mh1,
				},
			},
			expCount:              1,
			expHandlersInOrderLen: 2,
		},
		{
			ID: testhelper.MkID(
				"two new handlers, remove last, trailing nil removed"),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: add,
					mh:          mh2,
				},
				{
					addOrRemove: remove,
					mh:          mh2,
				},
			},
			expCount:              1,
			expHandlersInOrderLen: 1,
		},
		{
			ID: testhelper.MkID(
				"three new handlers, remove all"),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: add,
					mh:          mh2,
				},
				{
					addOrRemove: add,
					mh:          mh3,
				},
				{
					addOrRemove: remove,
					mh:          mh2,
				},
				{
					addOrRemove: remove,
					mh:          mh1,
				},
				{
					addOrRemove: remove,
					mh:          mh3,
				},
			},
			expCount:              0,
			expHandlersInOrderLen: 0,
		},
		{
			ID:     testhelper.MkID("one new handler, remove a different one"),
			ExpErr: testhelper.MkExpErr(errHandlerNotInSet.Error()),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: remove,
					mh:          mh2,
				},
			},
			expCount:              1,
			expHandlersInOrderLen: 1,
		},
		{
			ID:     testhelper.MkID("add same handler twice"),
			ExpErr: testhelper.MkExpErr(errHandlerAlreadyAdded.Error()),
			work: []addRemove{
				{
					addOrRemove: add,
					mh:          mh1,
				},
				{
					addOrRemove: add,
					mh:          mh1,
				},
			},
			expCount:              1,
			expHandlersInOrderLen: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			hs := newHandlerSet()

			var err error

		WorkLoop:
			for i, ar := range tc.work {
				switch ar.addOrRemove {
				case add:
					if err = hs.addHandler(ar.mh); err != nil {
						break WorkLoop
					}
				case remove:
					if err = hs.removeHandler(ar.mh); err != nil {
						break WorkLoop
					}
				default:
					t.Fatalf("%s: bad work entry at %d\n", tc.ID, i)
				}
			}

			testhelper.CheckExpErr(t, err, tc)
			testhelper.DiffInt(t, tc.IDStr(), "handlerSet entry count",
				hs.handlerCount(), tc.expCount)
			testhelper.DiffInt(t, tc.IDStr(), "handlerSet in-order entry count",
				len(hs.handlersInOrder), tc.expHandlersInOrderLen)
		})
	}
}
