package pusuclt

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	reflect "reflect"
	"sync"
	"time"

	"github.com/nickwells/pusu.mod/pusu"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errNoConn = errors.New("the client is not connected to the server")

// MsgHandler is a function that will be called when a Publish message is
// received over the connection.
//
// Note that the handler is called by the message reading goroutine. You are
// advised to pass any work off to a separate goroutine as soon as feasible
// so as to not delay the message reading goroutine.
type MsgHandler func(topic pusu.Topic, payload []byte)

// id returns the id of the MsgHandler
func (mh MsgHandler) id() uintptr {
	if mh == nil {
		return 0
	}

	return uintptr(reflect.ValueOf(mh).Pointer())
}

// topicHandlerMap maps between a Topic and the collection of MsgHandlers
// associated with it
type topicHandlerMap map[pusu.Topic]*handlerSet

// callbackMap maps between a message id and the Callback requested for that
// message
type callbackMap map[pusu.MsgID]Callback

// Client contains the client connection to a pub/sub server and the details
// needed to establish it.
type Client struct {
	mtx sync.Mutex

	cci *ConnInfo

	clientID string
	progName string

	namespace pusu.Namespace // namespace for all Publications and Subscriptions

	conn      io.ReadWriteCloser // the network connection
	connected bool               // flag set after connection is established

	handlers     topicHandlerMap    // the handler funcs for Publish messages
	sendChan     chan *pusu.Message // channel to send messages to the server
	stopChan     chan struct{}      // channel to disconnect from the server
	msgID        pusu.MsgID         // the next message id to use
	callbacks    callbackMap        // the callback for the message
	startTimeout time.Duration      // wait this long before aborting Startup

	tlsConfig *tls.Config
	logger    *slog.Logger
}

// nextMsgID increments and returns the message id
func (c *Client) nextMsgID() pusu.MsgID {
	c.msgID++

	return c.msgID
}

// NewClient creates an instance of a Conn and connects to the server given in
// the ConnInfo argument (info). If the connection fails it will return a nil
// Client pointer and a non-nil error. If the connection succeeds it will
// return a non-nil Client pointer and a nil error.
//
// The namespace argument is the namespace to which all the topics (when
// Publishing, Subscribing or Unsubscibing) will belong. All cooperating
// programs must use the same namespace as messages will only be exchanged
// between programs using the same namespace. Note that the publish/subscribe
// server may be configured to restrict the namespaces allowed.
//
// The progName is used to construct the client ID to be sent to the
// publish/subscribe server.
//
// The logger is used to record log messages.
//
// The info argument holds the connection information needed to make the
// connection.
func NewClient(
	namespace pusu.Namespace,
	progName string,
	logger *slog.Logger,
	info *ConnInfo,
) (*Client, error) {
	c := makeClient(namespace, progName, logger, info)

	if err := c.connect(); err != nil {
		return nil, err
	}

	return c, nil
}

// makeClient generates a new Client structure ready for it to make a
// connection. This is split out from the connection making code to simplify
// testing.
func makeClient(
	namespace pusu.Namespace,
	progName string,
	logger *slog.Logger,
	info *ConnInfo,
) *Client {
	return &Client{
		namespace: namespace,
		clientID:  makeClientID(progName),
		progName:  progName,
		logger: logger.With(
			pusu.NetAddressAttr(info.SvrAddress),
			namespace.Attr()),
		cci:          info,
		startTimeout: time.Second,
		handlers:     make(topicHandlerMap),
		callbacks:    make(callbackMap),
	}
}

// serverDetails returns a string giving a standard description of the
// pub/sub server the client is connecting to.
func (c *Client) serverDetails() string {
	return fmt.Sprintf("pub/sub server (%q)", c.cci.SvrAddress)
}

// connect connects to the addressed server. It returns an error if anything
// goes wrong. If it returns a nil error the connection was correctly
// established.
func (c *Client) connect() error {
	c.logger.Info("Connecting")

	if err := c.cci.CertInfo.PopulateCert(); err != nil {
		return err
	}

	if err := c.cci.CertInfo.PopulateCertPool(); err != nil {
		return err
	}

	c.tlsConfig = &tls.Config{
		RootCAs:      c.cci.CertInfo.CertPool(),
		Certificates: []tls.Certificate{c.cci.CertInfo.Cert()},
		MinVersion:   tls.VersionTLS13,
	}

	var err error

	c.conn, err = tls.DialWithDialer(
		&net.Dialer{
			Timeout: c.cci.ConnTimeout,
		}, "tcp", c.cci.SvrAddress, c.tlsConfig)
	if err != nil {
		return fmt.Errorf("couldn't connect to %s: %w", c.serverDetails(), err)
	}

	c.sendChan = make(chan *pusu.Message)
	c.stopChan = make(chan struct{})

	c.connected = true

	go c.run()
	go c.readConn()

	var startAckChan chan error

	if startAckChan, err = c.writeStartMsg(); err != nil {
		_ = c.conn.Close()
		return fmt.Errorf("client startup failed: %s: %w",
			c.serverDetails(), err)
	}

	err = c.startCheck(startAckChan)

	c.logger.Info("Connected", pusu.ErrorAttr(err))

	return err
}

// writeStartMsg writes the identifying message to the connection. This
// should only be called once on the connection where it should be the first
// message sent.
func (c *Client) writeStartMsg() (chan error, error) {
	c.logger.Info("sending the start message")

	payload, err := proto.Marshal(&pusu.StartMsgPayload{
		ProtocolVersion: pusu.CurrentProtoVsn,
		ClientId:        c.clientID,
		Namespace:       string(c.namespace),
	})
	if err != nil {
		return nil, fmt.Errorf("could not marshal the Start message: %w", err)
	}

	msgID := c.nextMsgID()
	startAckChan := make(chan error)

	c.addCallback(msgID,
		func(err error) {
			startAckChan <- err
		})

	sm := pusu.Message{
		MT:      pusu.Start,
		MsgID:   msgID,
		Payload: payload,
	}

	err = sm.Write(c.conn)

	return startAckChan, err
}

// startCheck waits for a timeout or for the startAckChan to receive a
// message (a nil error if all went well, non-nil otherwise).
func (c *Client) startCheck(startAckChan chan error) error {
	timeout := time.NewTicker(c.startTimeout)
	defer timeout.Stop()

	var err error

	select {
	case err = <-startAckChan:
	case <-timeout.C:
		err = errors.New("the client startup timed-out")
	}

	return err
}

// Disconnect will cause the client to disconnect from the server. It returns
// a non-nil error if the client is not connected.
//
// Note that it is not possible to reconnect to the publish/subscribe server;
// a new client should be created.
func (c *Client) Disconnect() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.connected {
		return errNoConn
	}

	c.stopChan <- struct{}{}

	return nil
}

// writePingMsg writes the ping message to the connection
func (c *Client) writePingMsg(t time.Time) error {
	payload, err := proto.Marshal(
		&pusu.PingMsgPayload{
			PingTime: timestamppb.New(t),
		})
	if err != nil {
		return fmt.Errorf("could not marshal the Ping message: %w", err)
	}

	pm := pusu.Message{
		MT:      pusu.Ping,
		Payload: payload,
	}

	if err := pm.Write(c.conn); err != nil {
		return fmt.Errorf("couldn't write the Ping message to %s: %w",
			c.serverDetails(), err)
	}

	return nil
}

// addHandler adds the TopicHandler to the client. it checks the TopicHandler
// first and reports any errors found. It returns true if this is the first
// handler for the topic.
func (c *Client) addHandler(th TopicHandler) (bool, error) {
	if err := th.check(); err != nil {
		return false, err
	}

	var hs *handlerSet

	var ok, newTopic bool

	if hs, ok = c.handlers[th.Topic]; !ok {
		newTopic = true
		hs = newHandlerSet()
	}

	if err := hs.addHandler(th.Handler); err != nil {
		return false, err
	}

	if newTopic {
		c.handlers[th.Topic] = hs
	}

	return newTopic, nil
}

// Subscribe causes a subscription message to be sent to the pub/sub
// server. All of the topics supplied are passed as subscriptions. Each
// handler is checked to make sure that the Topic passes the topic check and
// that the Handler is not nil. If either check fails then an error is
// returned and none of the subscriptions are made.
//
// Note that the message function associated with each topic will be called
// in a separate goroutine when a Publish message is received.
//
// Note that the Callback argument can be nil in which case it will be
// ignored. See the documentation for the Callback type to understand how it
// should be used.
func (c *Client) Subscribe(cb Callback, handlers ...TopicHandler) error {
	if len(handlers) == 0 {
		return nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.connected {
		return errNoConn
	}

	smp := pusu.SubscriptionMsgPayload{}

	for i, th := range handlers {
		if newTopic, err := c.addHandler(th); err != nil {
			return fmt.Errorf("cannot add the handler for Topic %q (%d): %w",
				th.Topic, i, err)
		} else if newTopic {
			smp.Subs = append(smp.Subs,
				&pusu.SubscriptionMsgPayload_Sub{Topic: string(th.Topic)})
		}
	}

	if len(smp.Subs) == 0 {
		// all the subscriptions previously existed - we are just adding new
		// handlers so don't send a message to the pub/sub server
		return nil
	}

	payload, err := proto.Marshal(&smp)
	if err != nil {
		c.logger.Error("could not marshal the Subscribe message",
			pusu.ErrorAttr(err))

		return fmt.Errorf("could not marshal the Subscribe message: %w", err)
	}

	msgID := c.nextMsgID()

	c.addCallback(msgID, cb)
	c.sendChan <- &pusu.Message{
		MT:      pusu.Subscribe,
		MsgID:   msgID,
		Payload: payload,
	}

	return nil
}

// Unsubscribe causes an unsubscription message to be sent to the pub/sub
// server. All of the topics supplied are unsubscribed from. Note that the
// Callback argument can be nil in which case it will be ignored. See the
// documentation for the Callback type to understand how it should be used.
func (c *Client) Unsubscribe(cb Callback, handlers ...TopicHandler) error {
	if len(handlers) == 0 {
		return nil
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.connected {
		return errNoConn
	}

	smp := pusu.SubscriptionMsgPayload{}

	for i, th := range handlers {
		var hs *handlerSet

		var ok bool

		if hs, ok = c.handlers[th.Topic]; !ok {
			return fmt.Errorf(
				"there is no existing subscription for Topic %q (%d)",
				th.Topic, i)
		}

		if err := hs.removeHandler(th.Handler); err != nil {
			return fmt.Errorf("cannot remove the handler for Topic %q (%d): %w",
				th.Topic, i, err)
		}

		if hs.handlerCount() == 0 {
			delete(c.handlers, th.Topic)
			smp.Subs = append(smp.Subs,
				&pusu.SubscriptionMsgPayload_Sub{Topic: string(th.Topic)})
		}
	}

	if len(smp.Subs) == 0 {
		// none of the subscriptions have had their last handler removed so
		// don't send a message to the pub/sub server
		return nil
	}

	payload, err := proto.Marshal(&smp)
	if err != nil {
		c.logger.Error("could not marshal the Unsubscribe message",
			pusu.ErrorAttr(err))

		return fmt.Errorf("could not marshal the Unsubscribe message: %w", err)
	}

	msgID := c.nextMsgID()

	c.addCallback(msgID, cb)
	c.sendChan <- &pusu.Message{
		MT:      pusu.Unsubscribe,
		MsgID:   msgID,
		Payload: payload,
	}

	return nil
}

// Publish causes a publication message to be sent to the pub/sub server. The
// topic is checked before being added and if it does not pass then an error
// is returned.
func (c *Client) Publish(
	cb Callback,
	topic pusu.Topic,
	payload []byte,
) error {
	if err := topic.Check(); err != nil {
		return err
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	pmp := pusu.PublishMsgPayload{
		Topic:   string(topic),
		Payload: payload,
	}

	msgPayload, err := proto.Marshal(&pmp)
	if err != nil {
		c.logger.Error("could not marshal the Publish message",
			pusu.ErrorAttr(err))

		return fmt.Errorf("could not marshal the Publish message: %w", err)
	}

	if !c.connected {
		return errNoConn
	}

	msgID := c.nextMsgID()

	c.addCallback(msgID, cb)
	c.sendChan <- &pusu.Message{
		MT:      pusu.Publish,
		MsgID:   msgID,
		Payload: msgPayload,
	}

	return nil
}

// close closes the connection to the pub/sub server
func (c *Client) close() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.connected {
		c.logger.Error("cannot close connection",
			pusu.ErrorAttr(errNoConn))
		return
	}

	c.connected = false
	close(c.sendChan)
	close(c.stopChan)

	c.logger.Info("closing the pub/sub server connection")

	if err := c.conn.Close(); err != nil {
		c.logger.Error("problem closing the pub/sub server connection",
			pusu.ErrorAttr(err))
	} else {
		c.logger.Info("pub/sub server connection closed")
	}
}

// isPingable returns true if the client is pingable. This is true if the
// client has a ping handler function and the ping interval has been set to
// some value greater than 0.
func (c *Client) isPingable() bool {
	return c.cci.pingHandler != nil &&
		c.cci.PingInterval > 0
}

// run runs the message loop. Ping messages are generated only if the client
// connection has a Ping handler function and the Ping interval is greater
// than zero.
func (c *Client) run() {
	defer c.close()

	c.logger.Info("connection running")

	pingTicker := &time.Ticker{}

	if c.isPingable() {
		pingTicker = time.NewTicker(c.cci.PingInterval)
		defer pingTicker.Stop()
	}

Loop:
	for {
		select {
		case <-c.stopChan:
			c.logger.Info("disconnecting")

			break Loop
		case msg := <-c.sendChan:
			if err := msg.Write(c.conn); err != nil {
				c.logger.Error(
					"couldn't write the message to the pub/sub server",
					msg.MT.Attr(),
					pusu.ErrorAttr(err))

				break Loop
			}

		case now := <-pingTicker.C:
			if c.cci.pingHandler == nil { // should never happen but ...
				c.logger.Error("unexpected Ping ticker event")
				continue Loop
			}

			if err := c.writePingMsg(now); err != nil {
				c.logger.Error(
					"couldn't ping the pub/sub server",
					pusu.ErrorAttr(err))

				break Loop
			}
		}
	}
}

// addCallback adds the passed Callback to the Conn's callbacks map if it is
// non-nil.
func (c *Client) addCallback(id pusu.MsgID, cb Callback) {
	if cb != nil {
		c.callbacks[id] = cb
	}
}

// getCallback reads the Callback from the callbacks map and if it was
// present if will remove the entry from callbacks and return it. Otherwise
// it will return nil.
func (c *Client) getCallback(id pusu.MsgID) Callback {
	if cb, ok := c.callbacks[id]; ok {
		delete(c.callbacks, id)

		return cb
	}

	return nil
}

// callback finds the callback entry in the callback map and if it was in the
// map it removes the map entry and calls the callback function in a new
// goroutine.
func (c *Client) callback(id pusu.MsgID, err error) {
	if cb := c.getCallback(id); cb != nil {
		go cb(err)
	}
}

// unMarshalErr constructs the error from the error message
func (c *Client) unMarshalErr(msg pusu.Message) error {
	if msg.MT != pusu.Error {
		return fmt.Errorf("cannot create an error from a message of type %s",
			msg.MT)
	}

	if msg.Payload == nil {
		return errors.New("no error was passed in the Error message")
	}

	var emp pusu.ErrorMsgPayload
	if err := proto.Unmarshal(msg.Payload, &emp); err != nil {
		return fmt.Errorf("could not unmarshal the Error message: %w", err)
	}

	return errors.New(emp.Error)
}

// readConn repeatedly reads from the connection and calls the message
// handler for each message read.
func (c *Client) readConn() {
	c.logger.Info("connection reading started")

Loop:
	for {
		msg, err := pusu.ReadMsg(c.conn)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				c.logger.Error("read failure on the connection",
					pusu.ErrorAttr(err))
			}

			break Loop
		}

		c.logger.Info("received", msg.MT.Attr())

		if err = c.handleMessageByType(msg); err != nil {
			c.logger.Error("message handling error",
				msg.MT.Attr(),
				pusu.ErrorAttr((err)))

			break Loop
		}
	}
	c.logger.Info("connection reading finished")
}

// handleMessageByType switches on the message type handling each type
// appropriately. It returns a non-nil error if any message is not properly
// handled.
func (c *Client) handleMessageByType(msg pusu.Message) error {
	var err error

	switch msg.MT {
	case pusu.Error:
		err = c.handleError(msg)
		c.callback(msg.MsgID, err)
	case pusu.Ack:
		c.callback(msg.MsgID, nil)
	case pusu.Publish:
		err = c.handlePublish(msg)
	case pusu.Ping:
		err = c.handlePing(msg)
	default:
		err = errors.New("protocol error - unexpected message")
	}

	return err
}

// handleError extracts the error from the message, logs it and returns it.
func (c *Client) handleError(msg pusu.Message) error {
	err := c.unMarshalErr(msg)

	c.logger.Error("a server error was received",
		msg.MsgID.Attr(),
		pusu.ErrorAttr(err))

	return err
}

// handlePublish extracts the Topic and PublishMsgPayload and calls any
// registered MsgHandlers on it.
func (c *Client) handlePublish(msg pusu.Message) error {
	var pubMsg pusu.PublishMsgPayload

	if err := msg.Unmarshal(&pubMsg, c.logger); err == nil {
		return err
	}

	c.callMsgHandlers(pusu.Topic(pubMsg.Topic), pubMsg.Payload)

	return nil
}

// handlePing tries to handle the Ping message returning an error if it could
// not do so.
func (c *Client) handlePing(msg pusu.Message) error {
	if c.cci.pingHandler == nil { // should never happen but ...
		c.logger.Error("unexpected Ping message received")
		return nil
	}

	pmp := pusu.PingMsgPayload{}
	if err := msg.Unmarshal(&pmp, c.logger); err != nil {
		return err
	}

	go c.cci.pingHandler(time.Since(pmp.PingTime.AsTime()))

	return nil
}

// callMsgHandlers will look up the message handlers for the topic and call
// them in the order they were registered.
func (c *Client) callMsgHandlers(t pusu.Topic, payload []byte) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if hs, ok := c.handlers[t]; ok {
		for _, h := range hs.handlersInOrder {
			if h != nil {
				h(t, payload)
			}
		}
	}
}
