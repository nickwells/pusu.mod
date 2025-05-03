package pusu

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// MaxMessagePayload is the maximum allowed size of the payload in a Message
const MaxMessagePayload = math.MaxUint16

// Message represents a message to or from a client
type Message struct {
	MT      MsgType // the message type
	Payload []byte  // the message payload
}

// Write will write the message to the writer. It will return an error if any
// of the writes fail or if the payload is greater than the MaxMessagePayload
// limit
func (m *Message) Write(w io.Writer) error {
	if len(m.Payload) > MaxMessagePayload {
		return fmt.Errorf("the message payload is too big: %d (max: %d)",
			len(m.Payload), MaxMessagePayload)
	}

	err := binary.Write(w, binary.LittleEndian, uint32(MagicID))
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, m.MT)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, uint16(len(m.Payload)))
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.LittleEndian, m.Payload)

	return nil
}

// readMsgIntro reads the first uint32 value from the reader. If the read
// fails or the value read is not equal to the MagicID it will return an
// error.
func readMsgIntro(r io.Reader) error {
	var msgIntro uint32

	err := binary.Read(r, binary.LittleEndian, &msgIntro)
	if err != nil {
		return err
	}

	if msgIntro != MagicID {
		return fmt.Errorf("invalid message intro should be: %X, is: %X",
			MagicID, msgIntro)
	}

	return nil
}

// readMsgType reads the message type from the reader. If the read fails or
// the value read is not a valid message type it will return an error.
func readMsgType(r io.Reader) (MsgType, error) {
	var mt MsgType

	err := binary.Read(r, binary.LittleEndian, &mt)
	if err != nil {
		return 0, err
	}

	if err = mt.Check(); err != nil {
		return 0, err
	}

	return mt, nil
}

// readMsgPayload reads first the payload size and then the message payload
// from the reader. If either read fails it will return an error.
func readMsgPayload(r io.Reader) ([]byte, error) {
	var msgLen uint16

	err := binary.Read(r, binary.LittleEndian, &msgLen)
	if err != nil {
		return nil, err
	}

	if msgLen == 0 {
		return []byte{}, nil
	}

	buf := make([]byte, msgLen)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadMsg will read the next message from the reader and returns a Message
// object from the reader. If any part of the reading returns an error this
// will fail and the error returned will be non-nil; the returned message
// will not be meaningful and the reader is no longer usable.
func ReadMsg(r io.Reader) (m Message, err error) {
	if err = readMsgIntro(r); err != nil {
		return m, err
	}

	m.MT, err = readMsgType(r)
	if err != nil {
		return m, err
	}

	m.Payload, err = readMsgPayload(r)
	if err != nil {
		return m, err
	}

	return m, nil
}
