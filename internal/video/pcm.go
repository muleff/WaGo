package video

// takePCM never blocks the realtime callback. Only consumed PCM advances time.
func takePCM(out []byte, carry *[]byte, queue <-chan []byte) int {
	off := 0
	for off < len(out) {
		if len(*carry) == 0 {
			select {
			case chunk, ok := <-queue:
				if !ok {
					clear(out[off:])
					return off
				}
				*carry = chunk
			default:
				clear(out[off:])
				return off
			}
		}
		n := copy(out[off:], *carry)
		*carry = (*carry)[n:]
		off += n
	}
	return off
}
