package def

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
	"github.com/golang/protobuf/proto"
)

func TestXx(t *testing.T) {
	x := make(chan int,1)
	x <- 10
	fmt.Println("xxxxxxxxxxxx")
	close(x)
}


func FlagCheck(mode Flag) {
	if mode&FlagEnd == 0 {
		panic("xxxx end")
	}
	if mode&FlagData != 0 {
		panic("xxxx data ")
	}

	fmt.Println("iiiiiiiiii")

}

func TestFlag(t *testing.T) {
	fmt.Println(FlagHead)
	fmt.Println(FlagData)
	fmt.Println(FlagSetting)
	fmt.Println(FlagEnd)
	x := FlagEnd|FlagHead
	FlagCheck(x)
}

func TestGoconvy(t *testing.T) {
	Convey("Equality assertions should be accessible", t, FailureContinues,func() {
		So(1, ShouldEqual, 1)
		time.Sleep(3*time.Second)
		So(1, ShouldNotEqual, 2)
		time.Sleep(3*time.Second)
		So(1, ShouldEqual, 1.000000000000001)
		time.Sleep(3*time.Second)
		So(1, ShouldNotAlmostEqual, 2, 0.5)
		time.Sleep(3*time.Second)
		req1 := &HelloReply{Message:"wangzun"}
		req2 := &HelloReply{Message:"wangzun"}
		So(proto.Equal(req1,req2),ShouldBeTrue)
		//Convey("xxxxx",func() {})
	})
}
