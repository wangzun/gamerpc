package def

import (
	"fmt"
	"testing"
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
