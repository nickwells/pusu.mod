package pusuclt

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/nickwells/pusu.mod/pusu"
	"github.com/nickwells/testhelper.mod/v2/testhelper"
	"google.golang.org/protobuf/proto"
)

const (
	testNamespaceName = "test-namespace-name"
	testProgName      = "test-prog-name"
	testSvrAddr       = "test-address"
	testLogLevel      = " level=INFO "

	testCloseString = "Close() Called"
)

var commonSlogAttrs = pusu.NetAddressAttrKey + "=" + testSvrAddr + " " +
	pusu.NamespaceAttrKey + "=" + testNamespaceName

// testReadWriteCloser adds a minimal Closer to an io.ReadWriter
type testReadWriteCloser struct {
	io.ReadWriter
	cBuf io.Writer
	cErr error
}

// Close implements the Close method
func (trwc testReadWriteCloser) Close() error {
	if trwc.cBuf != nil {
		_, _ = trwc.cBuf.Write([]byte(testCloseString))
	}

	return trwc.cErr
}

func makeTestClient(
	loggerBuf, connBuf, closerBuf *bytes.Buffer, cErr error,
) *Client {
	testLogger := slog.New(slog.NewTextHandler(loggerBuf, nil))
	info := NewConnInfo(nil)
	info.SvrAddress = testSvrAddr
	cc := makeClient(testNamespaceName, testProgName, testLogger, info)

	if connBuf == nil {
		cc.connected = false
	} else {
		cc.conn = testReadWriteCloser{
			ReadWriter: connBuf,
			cBuf:       closerBuf,
			cErr:       cErr,
		}
		cc.sendChan = make(chan *pusu.Message)
		cc.stopChan = make(chan struct{})

		cc.handlers = make(topicHandlerMap)
		cc.callbacks = make(callbackMap)

		cc.connected = true
	}

	return cc
}

func TestMakeClientID(t *testing.T) {
	logMsg := "test-message"
	tc := struct {
		testhelper.ID
		testhelper.ExpSlogMsgList
	}{
		ID: testhelper.MkID("Conn.MakeClientID"),
		ExpSlogMsgList: testhelper.MkExpSlogMsgList(
			testhelper.MkExpSlogMsg(slog.LevelInfo,
				slog.MessageKey+"="+logMsg,
				commonSlogAttrs),
		),
	}
	loggerBuf := &bytes.Buffer{}

	cc := makeTestClient(loggerBuf, nil, nil, nil)

	testhelper.DiffString(t, tc.Name, "namespace",
		cc.namespace, testNamespaceName)
	testhelper.DiffString(t, tc.Name, "program name",
		cc.progName, testProgName)
	testhelper.DiffString(t, tc.Name, "server details",
		cc.serverDetails(),
		`pub/sub server ("`+testSvrAddr+`")`)

	cc.logger.Info("test-message")

	testhelper.CheckExpSlogMessages(t, loggerBuf.String(), tc)
}

func TestConnWriteStartMsg(t *testing.T) {
	tc := struct {
		testhelper.ID
		testhelper.ExpErr
		testhelper.ExpSlogMsgList
	}{
		ID: testhelper.MkID("Conn.writeStartMsg"),
		ExpSlogMsgList: testhelper.MkExpSlogMsgList(
			testhelper.MkExpSlogMsg(slog.LevelInfo,
				slog.MessageKey+`="sending the start message"`,
				commonSlogAttrs,
			),
		),
	}
	loggerBuf := &bytes.Buffer{}
	connBuf := &bytes.Buffer{}

	cc := makeTestClient(loggerBuf, connBuf, nil, nil)

	var startAckChan chan error

	var err error

	startAckChan, err = cc.writeStartMsg()
	testhelper.CheckExpErr(t, err, tc)
	testhelper.CheckExpSlogMessages(t, loggerBuf.String(), tc)

	go func() {
		err := cc.startCheck(startAckChan)

		testhelper.CheckExpErr(t, err, tc)
	}()

	connMessage, err := pusu.ReadMsg(connBuf)
	testhelper.CheckExpErr(t, err, tc)
	cc.callback(cc.msgID-1, nil) // simulate the receipt of the Ack/Err

	if err == nil {
		var startMsg pusu.StartMsgPayload

		err = proto.Unmarshal(connMessage.Payload, &startMsg)
		testhelper.CheckExpErr(t, err, tc)

		if err == nil {
			testhelper.DiffString(t, tc.IDStr(), "namespace",
				startMsg.Namespace, testNamespaceName)
		}
	}
}

func TestConnDisconnect(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		loggerBuf *bytes.Buffer
		connBuf   *bytes.Buffer
	}{
		{
			ID:        testhelper.MkID("Conn.Disconnect - connected"),
			loggerBuf: &bytes.Buffer{},
			connBuf:   &bytes.Buffer{},
		},
		{
			ID:        testhelper.MkID("Conn.Disconnect - not connected"),
			ExpErr:    testhelper.MkExpErr(errNoConn.Error()),
			loggerBuf: &bytes.Buffer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cc := makeTestClient(tc.loggerBuf, tc.connBuf, nil, nil)

			go func() { <-cc.stopChan }()

			err := cc.Disconnect()

			testhelper.CheckExpErr(t, err, tc)
		})
	}
}

func TestConnClose(t *testing.T) {
	const closerErrText = "test-error"

	testCases := []struct {
		testhelper.ID
		testhelper.ExpSlogMsgList
		loggerBuf *bytes.Buffer
		connBuf   *bytes.Buffer
		closerBuf *bytes.Buffer
		closerErr error
	}{
		{
			ID:        testhelper.MkID("Conn.close - connected"),
			loggerBuf: &bytes.Buffer{},
			connBuf:   &bytes.Buffer{},
			closerBuf: &bytes.Buffer{},
			ExpSlogMsgList: testhelper.MkExpSlogMsgList(
				testhelper.MkExpSlogMsg(slog.LevelInfo,
					slog.MessageKey+"="+
						`"closing the pub/sub server connection"`,
					commonSlogAttrs,
				),
				testhelper.MkExpSlogMsg(slog.LevelInfo,
					slog.MessageKey+"="+`"pub/sub server connection closed"`,
					commonSlogAttrs,
				),
			),
		},
		{
			ID: testhelper.MkID(
				"Conn.close - connected - bad Close"),
			loggerBuf: &bytes.Buffer{},
			connBuf:   &bytes.Buffer{},
			closerBuf: &bytes.Buffer{},
			closerErr: errors.New(closerErrText),
			ExpSlogMsgList: testhelper.MkExpSlogMsgList(
				testhelper.MkExpSlogMsg(slog.LevelInfo,
					slog.MessageKey+"="+
						`"closing the pub/sub server connection"`,
					commonSlogAttrs,
				),
				testhelper.MkExpSlogMsg(slog.LevelError,
					slog.MessageKey+"="+
						`"problem closing the pub/sub server connection"`,
					commonSlogAttrs,
					pusu.ErrorAttrKey+"="+closerErrText,
				),
			),
		},
		{
			ID: testhelper.MkID("Conn.close - not connected"),
			ExpSlogMsgList: testhelper.MkExpSlogMsgList(
				testhelper.MkExpSlogMsg(slog.LevelError,
					slog.MessageKey+"="+`"cannot close connection"`,
					commonSlogAttrs,
					pusu.ErrorAttrKey+"="+`"`+errNoConn.Error()+`"`),
			),
			loggerBuf: &bytes.Buffer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cc := makeTestClient(
				tc.loggerBuf, tc.connBuf, tc.closerBuf, tc.closerErr)
			cc.close()
			testhelper.DiffBool(t, tc.IDStr(), "connected flag",
				cc.connected, false)
			testhelper.CheckExpSlogMessages(t, tc.loggerBuf.String(), tc)
		})
	}
}

func TestConnWritePingMsg(t *testing.T) {
	testCases := []struct {
		testhelper.ID
		testhelper.ExpErr
		loggerBuf *bytes.Buffer
		connBuf   *bytes.Buffer
	}{
		{
			ID:        testhelper.MkID("Conn.writePingMsg"),
			loggerBuf: &bytes.Buffer{},
			connBuf:   &bytes.Buffer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cc := makeTestClient(tc.loggerBuf, tc.connBuf, nil, nil)

			err := cc.writePingMsg(time.Now())
			testhelper.CheckExpErr(t, err, tc)

			connMessage, err := pusu.ReadMsg(tc.connBuf)
			testhelper.CheckExpErr(t, err, tc)

			if err == nil {
				var pingMsg pusu.PingMsgPayload

				err = proto.Unmarshal(connMessage.Payload, &pingMsg)
				testhelper.CheckError(t, tc.IDStr(), err, false, nil)

				if err == nil {
					testhelper.DiffTimeApprox(t, tc.IDStr(), "ping time",
						pingMsg.PingTime.AsTime(), time.Now(), time.Second)
				}
			}
		})
	}
}

// makeTestMsgHandler returns a MsgHandler that will record when it is called
// in the supplied buf.
func makeTestMsgHandler(buf *bytes.Buffer, handlerNum int) MsgHandler {
	return func(topic pusu.Topic, payload []byte) {
		fmt.Fprintf(buf, "%s:%d=%s\n",
			topic, handlerNum, string(payload))
	}
}

func TestConnAddHandler(t *testing.T) {
	msgHandlerBuf1 := &bytes.Buffer{}
	mh1 := makeTestMsgHandler(msgHandlerBuf1, 0)
	msgHandlerBuf2 := &bytes.Buffer{}
	mh2 := makeTestMsgHandler(msgHandlerBuf2, 0)

	// THWithErr bundles TopicHandlers with the expected return from the
	// addHandler method
	type THWithErr struct {
		testhelper.ExpErr
		th          TopicHandler
		expNewTopic bool
	}

	testCases := []struct {
		testhelper.ID
		loggerBuf *bytes.Buffer
		handlers  []THWithErr
	}{
		{
			ID:        testhelper.MkID("bad TopicHandler - bad topic"),
			loggerBuf: &bytes.Buffer{},
			handlers: []THWithErr{
				{
					ExpErr: testhelper.MkExpErr(
						`bad topic "bad-topic" - it must start with a '/'`),
					th: TopicHandler{
						Topic:   "bad-topic",
						Handler: nil,
					},
					expNewTopic: false,
				},
			},
		},
		{
			ID:        testhelper.MkID("bad TopicHandler - nil handler"),
			loggerBuf: &bytes.Buffer{},
			handlers: []THWithErr{
				{
					ExpErr: testhelper.MkExpErr(
						`the MsgHandler for Topic "/topic" is nil`),
					th: TopicHandler{
						Topic:   "/topic",
						Handler: nil,
					},
					expNewTopic: false,
				},
			},
		},
		{
			ID:        testhelper.MkID("good TopicHandler - one"),
			loggerBuf: &bytes.Buffer{},
			handlers: []THWithErr{
				{
					th: TopicHandler{
						Topic:   "/topic",
						Handler: mh1,
					},
					expNewTopic: true,
				},
			},
		},
		{
			ID:        testhelper.MkID("bad duplicate TopicHandler"),
			loggerBuf: &bytes.Buffer{},
			handlers: []THWithErr{
				{
					th: TopicHandler{
						Topic:   "/topic",
						Handler: mh1,
					},
					expNewTopic: true,
				},
				{
					ExpErr: testhelper.MkExpErr(
						"the handler has already been added"),
					th: TopicHandler{
						Topic:   "/topic",
						Handler: mh1,
					},
					expNewTopic: false,
				},
			},
		},
		{
			ID: testhelper.MkID(
				"good TopicHandler - different handlers, same topic"),
			loggerBuf: &bytes.Buffer{},
			handlers: []THWithErr{
				{
					th: TopicHandler{
						Topic:   "/topic",
						Handler: mh1,
					},
					expNewTopic: true,
				},
				{
					th: TopicHandler{
						Topic:   "/topic",
						Handler: mh2,
					},
					expNewTopic: false,
				},
			},
		},
		{
			ID: testhelper.MkID(
				"good TopicHandler - same handlers, different topic"),
			loggerBuf: &bytes.Buffer{},
			handlers: []THWithErr{
				{
					th: TopicHandler{
						Topic:   "/topic",
						Handler: mh1,
					},
					expNewTopic: true,
				},
				{
					th: TopicHandler{
						Topic:   "/topic2",
						Handler: mh1,
					},
					expNewTopic: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cc := makeTestClient(tc.loggerBuf, nil, nil, nil)

			var ccTopicCount, ccHandlerCount int

			for i, tcth := range tc.handlers {
				newTopic, err := cc.addHandler(tcth.th)

				if newTopic {
					ccTopicCount++
				}

				if err == nil {
					ccHandlerCount++
				}

				testFailed := false
				if testhelper.DiffBool(t,
					tc.IDStr(), "newTopic",
					newTopic, tcth.expNewTopic) {
					testFailed = true
				}

				if !testhelper.CheckExpErrWithID(t, tc.IDStr(), err, tcth) {
					testFailed = true
				}

				if testFailed {
					t.Errorf("%s: unexpected failure at handler %d: %s",
						tc.IDStr(), i, tcth.th)
				}
			}

			testhelper.DiffInt(t,
				tc.IDStr(), "topic count",
				len(cc.handlers), ccTopicCount)

			actHandlerCount := 0
			for _, handlers := range cc.handlers {
				actHandlerCount += len(handlers.handlersInOrder)
			}

			testhelper.DiffInt(t,
				tc.IDStr(), "handler count",
				actHandlerCount, ccHandlerCount)
		})
	}
}

// topicBuf is used in the MsgHandler funcs, below, and in
// TestConnHandleMessage, below.
var topicBuf map[pusu.Topic]*bytes.Buffer

// Note that the following funcs (which you might think could be generated
// with makeTestMsgHandler above) must be declared separately otherwise they
// all get the same id. This is not a problem with functions generated
// outside a loop

func mhFunc0(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 0, string(payload))
}

func mhFunc1(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 1, string(payload))
}

func mhFunc2(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 2, string(payload))
}

func mhFunc3(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 3, string(payload))
}

func mhFunc4(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 4, string(payload))
}

func mhFunc5(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 5, string(payload))
}

func mhFunc6(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 6, string(payload))
}

func mhFunc7(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 7, string(payload))
}

func mhFunc8(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 8, string(payload))
}

func mhFunc9(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 9, string(payload))
}

func mhFunc10(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 10, string(payload))
}

func mhFunc11(topic pusu.Topic, payload []byte) {
	fmt.Fprintf(topicBuf[topic], "%s:%d=%s\n", topic, 11, string(payload))
}

var handlerFuncs = []MsgHandler{
	mhFunc0,
	mhFunc1,
	mhFunc2,
	mhFunc3,
	mhFunc4,
	mhFunc5,
	mhFunc6,
	mhFunc7,
	mhFunc8,
	mhFunc9,
	mhFunc10,
	mhFunc11,
}

func TestConnHandleMessage(t *testing.T) {
	type TopicAction struct {
		t      pusu.Topic
		action string
	}

	testCases := []struct {
		testhelper.ID
		loggerBuf       *bytes.Buffer
		topicActions    []TopicAction
		expectedResults map[pusu.Topic]string
	}{
		{
			ID:        testhelper.MkID("1 sub, 1 pub"),
			loggerBuf: &bytes.Buffer{},
			topicActions: []TopicAction{
				{t: "/topicA", action: "sub"},
				{t: "/topicA", action: "pub"},
			},
			expectedResults: map[pusu.Topic]string{
				"/topicA": "/topicA:0=1\n",
			},
		},
		{
			ID:        testhelper.MkID("4 sub, 3 pub"),
			loggerBuf: &bytes.Buffer{},
			topicActions: []TopicAction{
				{t: "/topicA", action: "sub"},
				{t: "/topicA", action: "pub"},
				{t: "/topicA", action: "sub"},
				{t: "/topicA", action: "pub"},
			},
			expectedResults: map[pusu.Topic]string{
				"/topicA": "/topicA:0=1\n/topicA:0=3\n/topicA:2=3\n",
			},
		},
	}

	for _, tc := range testCases {
		if len(tc.topicActions) > len(handlerFuncs) {
			t.Log(tc.IDStr())
			t.Logf("\t: the number of     topicActions = %d",
				len(tc.topicActions))
			t.Logf("\t: the number of handlerFunctions = %d", len(handlerFuncs))
			t.Fatalf("bad test")
		}

		t.Run(tc.Name, func(t *testing.T) {
			cc := makeTestClient(tc.loggerBuf, nil, nil, nil)
			topicBuf = make(map[pusu.Topic]*bytes.Buffer)

			for i, ta := range tc.topicActions {
				if ta.action == "sub" {
					if _, ok := topicBuf[ta.t]; !ok {
						topicBuf[ta.t] = &bytes.Buffer{}
					}

					th := TopicHandler{
						Topic:   ta.t,
						Handler: handlerFuncs[i],
					}

					_, err := cc.addHandler(th)
					testhelper.CheckError(t, tc.IDStr(), err, false, nil)
				} else {
					cc.callMsgHandlers(ta.t, fmt.Append(nil, i))
				}
			}

			for topic, buf := range topicBuf {
				testhelper.DiffString(t,
					tc.IDStr(), "call record for "+string(topic),
					buf.String(), tc.expectedResults[topic])
			}
		})
	}
}
