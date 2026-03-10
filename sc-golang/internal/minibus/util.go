package minibus

// DropExcess will wrap a channel to break backpressure between its input and output.
// It will drop incoming messages when the consumer can't process them fast enough.
// When the consumer receives, it will always get the most recent message sent by the producer.
func DropExcess(in <-chan any) <-chan any {
	out := make(chan any)
	go func() {
		defer close(out)
		var message any
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
