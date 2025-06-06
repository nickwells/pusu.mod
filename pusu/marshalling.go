package pusu

import (
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/proto"
)

// logNonNilErr returns a function that will check the error it is passed and
// if it is non-nil, it will log it to the supplied logger with the supplied
// message.
func logNonNilErr(logger *slog.Logger, msg string) func(error) {
	return func(err error) {
		if err != nil {
			logger.Error(msg, ErrorAttr(err))
		}
	}
}

// Unmarshal takes the pusu.Message payload and unmarshals it into the
// protobuf message m (a type satisfying the proto.Message interface). If
// this returns an error it will be reported on the logger and a wrapped
// error returned. It also returns (and reports) an error if the payload is
// empty.
func (m *Message) Unmarshal(protoM proto.Message, logger *slog.Logger,
) (err error) {
	defer logNonNilErr(logger, "Unmarshalling failure")(err)

	if len(m.Payload) == 0 {
		err = fmt.Errorf("nothing to Unmarshal(...): MT: %s", m.MT)
	} else if err = proto.Unmarshal(m.Payload, protoM); err != nil {
		err = fmt.Errorf("could not Unmarshal(...): MT: %s: %w", m.MT, err)
	}

	return err
}

// Marshal takes the protobuf message (a type satisfying the proto.Message
// interface) and marshals it into the pusu.Message.Payload. If this returns
// an error it will be reported on the logger and a wrapped error returned.
func (m *Message) Marshal(protoM proto.Message, logger *slog.Logger,
) (err error) {
	defer logNonNilErr(logger, "Marshalling failure")(err)

	m.Payload, err = proto.Marshal(protoM)
	if err != nil {
		err = fmt.Errorf("could not Marshal(...): MT: %s: %w", m.MT, err)
	}

	return err
}
