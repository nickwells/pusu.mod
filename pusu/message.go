package pusu

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	// MaxMessagePayload is the maximum allowed size of the payload in a Message
	MaxMessagePayload = math.MaxUint16
	// magicID is the introductory value at the start of every message. It is
	// used to check for corrupted messages
	magicID = uint32(0xB0AD1CEA)
	// writeErrIntro is the start of every error message from the Write method
	writeErrIntro = "pusu.Message.Write failure: "
	// readErrIntro is the start of every error message from the Read method
	readErrIntro = "pusu.ReadMsg failure: "
)

// Message represents a message between a pub/sub client and server
type Message struct {
	MT      MsgType // the message type
	MsgID   MsgID   // the message ID
	Payload []byte  // the message payload
}

// msgHdr holds the common fixed part of a message, as written to or read
// from the connection
type msgHdr struct {
	Magic       uint32
	MT          MsgType
	MsgID       MsgID
	PayloadSize uint16
}

// writeLE writes the binary representation of the data to the Writer in
// LittleEndian order returning any errors
func writeLE(w io.Writer, data any) error {
	return binary.Write(w, binary.LittleEndian, data)
}

// Write will write the message to the writer. It will return an error if the
// payload is greater than the MaxMessagePayload limit, if the message type
// is invalid, or if any of the writes fail
func (m *Message) Write(w io.Writer) error {
	if len(m.Payload) > MaxMessagePayload {
		return fmt.Errorf(writeErrIntro+"bad payload - too big: %d (max: %d)",
			len(m.Payload), MaxMessagePayload)
	}

	if err := m.MT.Check(); err != nil {
		return fmt.Errorf(writeErrIntro+"%w", err)
	}

	hdr := msgHdr{
		Magic:       magicID,
		MT:          m.MT,
		MsgID:       m.MsgID,
		PayloadSize: uint16(len(m.Payload)), //nolint:gosec
	}

	if err := writeLE(w, hdr); err != nil {
		return fmt.Errorf(
			writeErrIntro+"could not write the message header: %w", err)
	}

	if hdr.PayloadSize == 0 {
		return nil
	}

	if err := writeLE(w, m.Payload); err != nil {
		return fmt.Errorf(writeErrIntro+"could not write the payload: %w", err)
	}

	return nil
}

// readMsgHdr reads the message header from the reader and checks the
// values. If the read fails or the checks fail it will return an error.
func readMsgHdr(r io.Reader) (msgHdr, error) {
	var hdr msgHdr

	err := binary.Read(r, binary.LittleEndian, &hdr)
	if err != nil {
		return hdr,
			fmt.Errorf(
				readErrIntro+"error while reading message header: %w",
				err)
	}

	if hdr.Magic != magicID {
		return hdr,
			fmt.Errorf(
				readErrIntro+"bad message start, should be: 0x%X, is: 0x%X",
				magicID, hdr.Magic)
	}

	if err := hdr.MT.Check(); err != nil {
		return hdr,
			fmt.Errorf(readErrIntro+"bad message type: %w", err)
	}

	return hdr, nil
}

// readMsgPayload reads first the payload size and then the message payload
// from the reader. If either read fails it will return an error.
func readMsgPayload(r io.Reader, buf []byte) error {
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf(
			readErrIntro+"error while reading message payload: %w",
			err)
	}

	return nil
}

// ReadMsg will read the next message from the reader and returns a Message
// object from the reader. If any part of the reading returns an error this
// will fail and the error returned will be non-nil; the returned message
// will not be meaningful and the reader is no longer usable.
func ReadMsg(r io.Reader) (Message, error) {
	var hdr msgHdr

	var msg Message

	var err error

	if hdr, err = readMsgHdr(r); err != nil {
		return msg, err
	}

	msg.MT = hdr.MT
	msg.MsgID = hdr.MsgID

	if hdr.PayloadSize > 0 {
		msg.Payload = make([]byte, hdr.PayloadSize)
		err = readMsgPayload(r, msg.Payload)
	}

	return msg, err
}
