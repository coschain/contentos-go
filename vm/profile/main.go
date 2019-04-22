package main

import (
	"flag"
	"fmt"
	"github.com/coschain/contentos-go/vm"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/go-interpreter/wagon/exec"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	for i := 0; i < 10000; i++ {
		simpleAdd()
	}
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}

func add(proc *exec.Process, a, b int32) int32 {
	return a + b
}

func simpleAdd() {
	wasmFile := "../testdata/add.wasm"
	data, _ := ioutil.ReadFile(wasmFile)
	ctx := vmcontext.Context{Code: data}
	cosVM := vm.NewCosVM(&ctx, nil, nil, nil)
	cosVM.Register("add", add, 3000)
	ret, err := cosVM.Run()
	fmt.Println(ret, err)
}
