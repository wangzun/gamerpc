package def

import "io"

type Flag uint8

const (
	FlagHead Flag     = 1 << iota
	FlagData
	FlagSetting
	FlagEnd
)

type Stream struct {
	id int32
	reader io.Reader
	recv   *recvBuffer
}
