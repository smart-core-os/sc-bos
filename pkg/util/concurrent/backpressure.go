package concurrent

import (
	"github.com/smart-core-os/sc-bos/pkg/util/bounded"
)

// BreakBackpressure will wrap a channel to break backpressure between its input and output.
// It will drop incoming messages when the consumer can't process them fast enough.
// When the consumer receives, it will always get the most recent message sent by the producer.
func BreakBackpressure[T any](in <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		var message T
		hasMessage := false
		for {
			if hasMessage {
				select {
				case newMessage, ok := <-in:
					if !ok {
						return
					}
					// replace the buffered message, discarding the old one
					message = newMessage

				case out <- message:
					// message sent successfully
					hasMessage = false
				}
			} else {
				newMessage, ok := <-in
				if !ok {
					return
				}
				message = newMessage
				hasMessage = true
			}
		}
	}()
	return out
}

func BreakBackpressureBuffered[T any](in <-chan T, size int) <-chan T {
	if size <= 0 {
		panic("invalid size")
	} else if size == 1 {
		return BreakBackpressure(in)
	}

	out := make(chan T)
	go func() {
		defer close(out)
		buffer := bounded.NewQueue[T](size)
		for {
			head, headOk := buffer.Peek()
			if headOk {
				select {
				case newMessage, ok := <-in:
					if !ok {
						return
					}
					_ = buffer.PushBack(newMessage)

				case out <- head:
					// message sent, remove the item we just peeked from the buffer
					_, _ = buffer.Pop()
				}
			} else {
				newMessage, ok := <-in
				if !ok {
					return
				}
				_ = buffer.PushBack(newMessage)
			}
		}
	}()
	return out
}
