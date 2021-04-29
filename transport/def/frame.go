package def

import (
	"encoding/binary"
	"errors"
)

//-------------------------------------------------------------------------------------------------------------------
// 帧结构
// uint16(frameLen) + uint32(streamId)+ uint8(flag) + uint8(send_type) + header(可选) + setting(可选) + data(可选)
//-------------------------------------------------------------------------------------------------------------------

const (
	FrameLen          = 2
	StreamIdLen       = 4
	FrameFlagLen      = 1
	SendTypeLen       = 1
	FramePreFix       = 8
)

const (
	FrameHeadLen    = 2
	FrameSettingLen = 2
)

type Flag uint8

const (
	FlagHead Flag     = 1 << iota
	FlagData
	FlagSetting
	FlagEnd
)

func (f Flag) Has(v Flag) bool {
	return (f & v) == v
}



const (
	CallReq  = 1
	CallRes  = 2
	Cast     = 3
)

const MaxFrameLen = 1<<15

type frameHead struct {
	framelen       uint16
	streamId       uint32
	flag           Flag
	typ            uint8
}

func getframeHeadFromBin(bin []byte) (*frameHead,error){
	if len(bin) != FramePreFix {
		return nil,errors.New("bin len err")
	}
	head := &frameHead{
		framelen: binary.BigEndian.Uint16(bin[0:2]),
		streamId: binary.BigEndian.Uint32(bin[2:6]),
		flag:Flag(bin[6]),
		typ :uint8(bin[7]),
	}
	return head,nil
}

func getframeHeadBin(head *frameHead,bin []byte){
	binary.BigEndian.PutUint16(bin,head.framelen)
	binary.BigEndian.PutUint32(bin,head.streamId)
	bin[6] = byte(head.flag)
	bin[7] = byte(head.typ)
}


