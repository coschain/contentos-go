package commands

import (
	"fmt"
	"time"
)

func autoTest () {
	time.Sleep(10 * time.Second)
	fmt.Println("mian func")
	for i:=0;i<len(globalObj.dposList);i++ {
		fmt.Println()
		fmt.Println()
		fmt.Println("main func active producers:   ", globalObj.dposList[i].ActiveProducers())
		fmt.Println()
		fmt.Println()
	}

	now := time.Now()
	globalObj.dposList[0].MaybeProduceBlock(now)
	globalObj.dposList[0].MaybeProduceBlock(now.Add( 3 * time.Second))
	globalObj.dposList[0].MaybeProduceBlock(now.Add( 6 * time.Second))
	time.Sleep(10*time.Second)
	fmt.Println("head block id:   ", globalObj.dposList[0].GetHeadBlockId())
	fmt.Println("head block id:   ", globalObj.dposList[1].GetHeadBlockId())
	fmt.Println("head block id:   ", globalObj.dposList[2].GetHeadBlockId())
}