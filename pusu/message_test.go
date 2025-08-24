package pusu

import (
	"bytes"
	"testing"

	"github.com/nickwells/testhelper.mod/v2/testhelper"
)

func TestMessageWrite(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		msg Message
	}{
		{
			ID: testhelper.MkID("good Write"),
			msg: Message{
				MT:      Start,
				Payload: []byte("hello"),
			},
		},
		{
			ID: testhelper.MkID("bad Write - payload too big"),
			ExpErr: testhelper.MkExpErr(
				writeErrIntro,
				"bad payload - too big: 65536 (max: 65535)"),
			msg: Message{
				MT:      Start,
				Payload: make([]byte, MaxMessagePayload+1),
			},
		},
		{
			ID: testhelper.MkID("bad Write - bad message type (>max)"),
			ExpErr: testhelper.MkExpErr(
				writeErrIntro,
				"bad message type",
				"too big"),
			msg: Message{
				MT:      MaxMsgType + 1,
				Payload: []byte("hello"),
			},
		},
		{
			ID: testhelper.MkID("bad Write - bad message type (<min)"),
			ExpErr: testhelper.MkExpErr(
				writeErrIntro,
				"bad message type: Invalid"),
			msg: Message{
				MT:      Invalid,
				Payload: []byte("hello"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var b bytes.Buffer

			err := tc.msg.Write(&b)
			testhelper.CheckExpErr(t, err, tc)

			if err == nil {
				msg, err := ReadMsg(&b)
				if err != nil {
					t.Log(tc.ID)
					t.Errorf(
						"\t: unexpected error reading the written message: %s",
						err)

					return
				}

				if msg.MT != tc.msg.MT {
					t.Log(tc.ID)
					t.Logf("\t: expected message type: %s\n", tc.msg.MT)
					t.Logf("\t:   actual message type: %s\n", msg.MT)
					t.Error("\t: bad read")
				}

				if testhelper.DiffSlice(t,
					tc.IDStr(), "payload",
					msg.Payload, tc.msg.Payload) {
					t.Error("\t: bad read")
				}
			}
		})
	}
}

// testMsgPart represents a named piece of data
type testMsgPart struct {
	name string
	data any
}

// makeBadBuf ...
func makeBadBuf(t *testing.T, msgParts ...testMsgPart) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	for _, part := range msgParts {
		if err := writeLE(buf, part.data); err != nil {
			t.Fatalf("couldn't write the %s: %s", part.name, err)
			return nil
		}
	}

	return buf
}

func TestMessageRead(t *testing.T) {
	goodMsgBuf := &bytes.Buffer{}
	emptyBuffer := &bytes.Buffer{}

	msg := &Message{
		MT:      Start,
		MsgID:   42,
		Payload: []byte("hello"),
	}

	if err := msg.Write(goodMsgBuf); err != nil {
		t.Fatal("couldn't write the message to buffer:", err)
	}

	badMagicBuf := makeBadBuf(t,
		testMsgPart{
			name: "bad magicID",
			data: msgHdr{
				Magic:       magicID + 1,
				MT:          Start,
				MsgID:       42,
				PayloadSize: 0,
			},
		})

	badMsgTypeBuf := makeBadBuf(t,
		testMsgPart{
			name: "bad header - MT: Invalid",
			data: msgHdr{
				Magic:       magicID,
				MT:          Invalid,
				MsgID:       42,
				PayloadSize: 0,
			},
		})

	zeroPayloadSizeBuf := makeBadBuf(t,
		testMsgPart{
			name: "good header - zero payload size",
			data: msgHdr{
				Magic:       magicID,
				MT:          Start,
				MsgID:       42,
				PayloadSize: 0,
			},
		})

	missingPayloadBuf := makeBadBuf(t,
		testMsgPart{
			name: "bad header - non-zero payload size, no payload",
			data: msgHdr{
				Magic:       magicID,
				MT:          Start,
				MsgID:       42,
				PayloadSize: 1,
			},
		})

	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		readBuf *bytes.Buffer
		expMsg  Message
	}{
		{
			ID: testhelper.MkID("bad message start - too few bytes"),
			ExpErr: testhelper.MkExpErr(
				readErrIntro,
				"error while reading message header: EOF"),
			readBuf: emptyBuffer,
		},
		{
			ID: testhelper.MkID("bad message start - bad magic number"),
			ExpErr: testhelper.MkExpErr(
				readErrIntro,
				"bad message start, should be: 0xb0ad1cea, is: 0xb0ad1ceb"),
			readBuf: badMagicBuf,
		},
		{
			ID: testhelper.MkID("bad message start - invalid message type"),
			ExpErr: testhelper.MkExpErr(
				readErrIntro,
				"bad message type: Invalid"),
			readBuf: badMsgTypeBuf,
		},
		{
			ID:      testhelper.MkID("good message - zero payload size"),
			readBuf: zeroPayloadSizeBuf,
		},
		{
			ID: testhelper.MkID("bad message start - missing payload"),
			ExpErr: testhelper.MkExpErr(
				readErrIntro,
				"error while reading message payload: EOF"),
			readBuf: missingPayloadBuf,
		},
		{
			ID:      testhelper.MkID("good message"),
			readBuf: goodMsgBuf,
		},
	}

	for _, tc := range testCases {
		if tc.readBuf == nil {
			continue
		}

		t.Run(tc.Name, func(t *testing.T) {
			_, err := ReadMsg(tc.readBuf)
			testhelper.CheckExpErr(t, err, tc)
		})
	}
}
