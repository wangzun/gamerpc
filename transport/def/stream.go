package def

import "io"

type Stream struct {
	id     uint32
	reader io.Reader
	recv   *recvBuffer
}
