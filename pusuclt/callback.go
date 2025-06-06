package pusuclt

// Callback is the type of a function that, if provided, will be called when
// an Ack or an Err is received for the message that was sent. It can be used
// to notify the caller when a message has been processed by the pub/sub
// server. Note that the Callback is always called in a separate goroutine so
// there is no guarantee of the order as it may be scheduled out of order by
// the Go runtime.
type Callback func(error)

// MakeCallback returns a Callback function that will send the supplied value
// on the supplied channel.
//
// Note that it is the caller's responsibility to ensure that there is some
// goroutine reading from the channel.
func MakeCallback[T any](c chan T, value T) Callback {
	return func(err error) {
		if err == nil {
			go func() {
				c <- value
			}()
		}
	}
}
