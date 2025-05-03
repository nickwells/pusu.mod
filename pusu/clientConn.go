package pusu

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/user"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MsgHandler is a function that will be called (in a separate thread) for
// every Publish message received over the connection.
type MsgHandler func(topic Topic, payload []byte)

// ClientConnInfo contains the client connection to a pub/sub server and the
// details needed to establish it.
type ClientConnInfo struct {
	name string // the name of the connection - it should reflect its purpose

	conn      *tls.Conn // the network connection
	address   string    // the network address for the pub/sub server
	connected bool      // a flag set when the connection has been established

	msgHandler MsgHandler    // the supplied handler func for Publish messages
	sendChan   chan *Message // channel to send messages over the connection
	abortChan  chan struct{} // channel for signaling to close the connection

	timeout      time.Duration // the connection dialler timeout
	pingInterval time.Duration // how long to wait between Pings

	clientID      string
	clientIDIsSet bool
	certInfo      CertInfo
	tlsConfig     *tls.Config
	logger        *slog.Logger
}

// NewClientConnInfo returns an instance of a ClientConnInfo with the correct
// defaults set. The supplied name should reflect the purpose of the
// connection, for instance "recording stats" or "primary"
func NewClientConnInfo(name string, handler MsgHandler) *ClientConnInfo {
	const (
		dfltConnTimeout  = 15
		dfltPingInterval = 2
	)

	return &ClientConnInfo{
		name:         name,
		timeout:      dfltConnTimeout * time.Second,
		pingInterval: dfltPingInterval * time.Second,
		clientID:     "[unset]",
		msgHandler:   handler,
	}
}

// SetID populates the client identity string that is sent to the pub/sub
// server in the first message.
func (cci *ClientConnInfo) SetID(progName string) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "[unknown]"
	}

	userdetails := "[unknown]"
	if user, err := user.Current(); err == nil {
		userdetails = user.Uid + "/" +
			user.Gid + "/" +
			user.Username + "(" + user.Name + ")"
	}

	cci.clientID = progName +
		";hostname: " + hostname +
		";user (uid/gid/name): " + userdetails +
		";pid: " + fmt.Sprintf("%d", os.Getpid())

	cci.clientIDIsSet = true
}

// ID returns the client ID (suitable for use in an Identify message)
func (cci *ClientConnInfo) ID() string {
	if !cci.clientIDIsSet {
		panic("the client ID has not been set")
	}
	return cci.clientID
}

// Connect connects to the addressed server. It returns false and logs a
// message if anything goes wrong. If it returns true the connection was
// correctly established.
func (cci *ClientConnInfo) Connect(
	logger *slog.Logger, handler MsgHandler,
) bool {
	cci.logger = logger.With(NetAddressAttr(cci.address))

	cci.logger.Info("connecting to the pub/sub server")

	if !cci.certInfo.PopulateCert(cci.logger) {
		return false
	}

	if !cci.certInfo.PopulateCertPool(cci.logger) {
		return false
	}

	cci.tlsConfig = &tls.Config{
		RootCAs:      cci.certInfo.CertPool(),
		Certificates: []tls.Certificate{cci.certInfo.Cert()},
		MinVersion:   tls.VersionTLS13,
	}

	var err error
	cci.conn, err = tls.DialWithDialer(
		&net.Dialer{
			Timeout: cci.timeout,
		}, "tcp", cci.address, cci.tlsConfig)
	if err != nil {
		cci.logger.Error("couldn't connect to the pub/sub server",
			ErrorAttr(err))

		return false
	}

	cci.logger.Info("connected to server")

	if err = cci.writeIdentityMsg(); err != nil {
		cci.logger.Error(
			"couldn't send the identity message to the pub/sub server",
			ErrorAttr(err))

		return false
	}

	cci.sendChan = make(chan *Message)
	cci.abortChan = make(chan struct{})
	cci.connected = true

	go cci.run()
	go cci.readConn()

	return true
}

// close closes the conection to the pub/sub server
func (cci *ClientConnInfo) close() {
	if !cci.connected {
		cci.logger.Error("cannot close the connection - not connected")
	}

	cci.logger.Info("closing the pub/sub server connection")

	if err := cci.conn.Close(); err != nil {
		cci.logger.Error("problem closing the pub/sub server connection",
			ErrorAttr(err))
	} else {
		cci.connected = false

		cci.logger.Info("pub/sub server connection closed")
	}
}

// writeIdentityMsg writes the identifying message to the connection. This
// should only be called once on the connection where it should be the first
// message sent.
func (cci *ClientConnInfo) writeIdentityMsg() error {
	cci.logger.Info("sending the identify message")

	id := []byte(cci.ID()) // panics if identity has not been set

	identMsg := Message{
		MT:      Identify,
		Payload: id,
	}

	return identMsg.Write(cci.conn)
}

// writePingMsg writes the ping message to the connection
func (cci *ClientConnInfo) writePingMsg(t time.Time) error {
	pmp := PingMsgPayload{}
	pmp.PingTime = timestamppb.New(t)

	payload, err := proto.Marshal(pmp.ProtoReflect().Interface())
	if err != nil {
		return fmt.Errorf("could not marshal the Ping message: %w", err)
	}

	pm := Message{
		MT:      Ping,
		Payload: payload,
	}

	if err := pm.Write(cci.conn); err != nil {
		return fmt.Errorf(
			"couldn't write the Ping message to the pub/sub server: %w",
			err)
	}

	return nil
}

// Subscribe causes a subscription message to be sent to the pub/sub
// server. All of the topics supplied are passed as subscriptions. Each topic
// is checked before being added and if any topic does not start with a '/'
// then an error is returned.
func (cci *ClientConnInfo) Subscribe(topic ...string) error {
	if !cci.connected {
		return errors.New("client has not connected to the server")
	}

	if len(topic) == 0 {
		return nil
	}

	smp := SubscriptionMsgPayload{}

	for _, t := range topic {
		if err := Topic(t).Check(); err != nil {
			return err
		}

		smp.Subs = append(smp.Subs, &SubscriptionMsgPayload_Sub{Topic: t})
	}

	payload, err := proto.Marshal(smp.ProtoReflect().Interface())
	if err != nil {
		cci.logger.Error("could not marshal the Subscribe message",
			ErrorAttr(err))

		return fmt.Errorf("could not marshal the Subscribe message: %w", err)
	}

	cci.sendChan <- &Message{
		MT:      Subscribe,
		Payload: payload,
	}

	return nil
}

// Unsubscribe causes an unsubscription message to be sent to the pub/sub
// server. All of the topics supplied are unsubscribed from.
func (cci *ClientConnInfo) Unsubscribe(topic ...string) error {
	if !cci.connected {
		return errors.New("client has not connected to the server")
	}

	if len(topic) == 0 {
		return nil
	}

	// Unsubscribe and Subscribe share the same payload
	smp := SubscriptionMsgPayload{}

	for _, t := range topic {
		smp.Subs = append(smp.Subs, &SubscriptionMsgPayload_Sub{Topic: t})
	}

	payload, err := proto.Marshal(smp.ProtoReflect().Interface())
	if err != nil {
		cci.logger.Error("could not marshal the Unsubscribe message",
			ErrorAttr(err))

		return fmt.Errorf("could not marshal the Unsubscribe message: %w", err)
	}

	cci.sendChan <- &Message{
		MT:      Unsubscribe,
		Payload: payload,
	}

	return nil
}

// Unsubscribe causes an unsubscription message to be sent to the pub/sub
// server. All of the topics supplied are unsubscribed from.
func (cci *ClientConnInfo) UnsubscribeAll() error {
	if !cci.connected {
		return errors.New("client has not connected to the server")
	}

	cci.sendChan <- &Message{
		MT: UnsubscribeAll,
	}

	return nil
}

// Publish causes a publication message to be sent to the pub/sub
// server. All of the topics supplied are passed as subscriptions. Each topic
// is checked before being added and if any topic does not start with a '/'
// then an error is returned.
func (cci *ClientConnInfo) Publish(topic string, publishPayload []byte) error {
	if !cci.connected {
		return errors.New("client has not connected to the server")
	}

	if err := Topic(topic).Check(); err != nil {
		return err
	}

	pmp := PublishMsgPayload{
		Topic:   topic,
		Payload: publishPayload,
	}

	payload, err := proto.Marshal(pmp.ProtoReflect().Interface())
	if err != nil {
		cci.logger.Error("could not marshal the Publish message",
			ErrorAttr(err))

		return fmt.Errorf("could not marshal the Publish message: %w", err)
	}

	m := Message{
		MT:      Publish,
		Payload: payload,
	}

	cci.sendChan <- &m

	return nil
}

// run runs the message loop
func (cci *ClientConnInfo) run() {
	defer cci.close()

	pingTicker := time.NewTicker(cci.pingInterval)
	defer pingTicker.Stop()

Loop:
	for {
		select {
		case msg := <-cci.sendChan:
			if err := msg.Write(cci.conn); err != nil {
				cci.logger.Error(
					"couldn't write the message to the pub/sub server",
					msg.MT.Attr(),
					ErrorAttr(err))

				break Loop
			}
		case <-cci.abortChan:
			cci.logger.Info("aborting the connection")

			break Loop

		case now := <-pingTicker.C:
			if err := cci.writePingMsg(now); err != nil {
				cci.logger.Error(
					"couldn't ping the pub/sub server",
					ErrorAttr(err))

				break Loop
			}
		}
	}
}

// readConn repeatedly reads from the connection and calls the message
// handler for each message read.
func (cci *ClientConnInfo) readConn() {
Loop:
	for {
		msg, err := ReadMsg(cci.conn)
		if err != nil {
			cci.logger.Error("read failure on the connection",
				ErrorAttr(err))

			break Loop
		}

		cci.logger.Info("received", msg.MT.Attr())

		switch msg.MT {
		case Publish:
			var pubMsg PublishMsgPayload
			if err := proto.Unmarshal(msg.Payload, &pubMsg); err != nil {
				cci.logger.Error("could not unmarshall the Publish message",
					ErrorAttr(err))

				break Loop
			}
			cci.msgHandler(Topic(pubMsg.Topic), pubMsg.Payload)
		case Ping:
			pmp := PingMsgPayload{}
			if err := proto.Unmarshal(msg.Payload, &pmp); err != nil {
				cci.logger.Error("could not unmarshall the Ping message",
					ErrorAttr(err))

				break Loop
			}
			cci.logger.Info("Ping response received",
				slog.Duration("Ping-Duration",
					time.Duration(time.Now().Sub(pmp.PingTime.AsTime()))))
		default:
			cci.logger.Error("protocol error - unexpected message",
				msg.MT.Attr())

			break Loop

		}
	}

	cci.abortChan <- struct{}{}
}
