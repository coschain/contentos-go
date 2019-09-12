package main

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc"
	grpcpb "github.com/coschain/contentos-go/rpc/pb"
	"os"
	"sort"
	"sync"
	"time"
)

var threadCount uint64
var endPoint string
var eachThreadRequest = 100
var name = constants.COSInitMiner
var wg = sync.WaitGroup{}

var countObject = &CountStruct{success:0,fail:0}
type CountStruct struct {
	success int
	fail    int
	sync.Mutex
}
func (c *CountStruct) addSuccess() {
	c.Lock()
	defer c.Unlock()
	c.success++
}
func (c *CountStruct) addFail() {
	c.Lock()
	defer c.Unlock()
	c.fail++
}

var timeObject = new(timeStruct)
type timeStruct struct {
	timeSlice []time.Duration
	sync.Mutex
}
func (t *timeStruct) addTimeInterval(tt time.Duration) {
	t.Lock()
	defer t.Unlock()
	t.timeSlice = append(t.timeSlice, tt)
}

type target []time.Duration
func (p target) Len() int {return len(p)}
func (p target) Less(i, j int) bool {return p[i] < p[j]}
func (p target) Swap(i, j int) {p[i], p[j] = p[j], p[i]}

var rootCmd = &cobra.Command{
	Use:   "readtest",
	Run:   startTest,
}

func main() {
	rootCmd.Flags().Uint64VarP(&threadCount, "thread", "t", 10, "./readtest -t=1000")
	rootCmd.Flags().StringVarP(&endPoint, "ip", "p", "127.0.0.1:8888", "./readtest -p=XXX")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startTest(cmd *cobra.Command, args []string) {
	conn, err := rpc.Dial(endPoint)
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	}
	rpcClient := grpcpb.NewApiServiceClient(conn)

	for i:=0;i<int(threadCount);i++ {
		wg.Add(1)
		go startEachThread(rpcClient)
	}
	wg.Wait()

	var total time.Duration
	sort.Sort(target(timeObject.timeSlice))
	length := len(timeObject.timeSlice)
	for i:=0;i<length;i++ {
		total += timeObject.timeSlice[i]
	}
	idx1 := int(float64(length)*0.5) - 1
	idx2 := int(float64(length)*0.66) - 1
	idx3 := int(float64(length)*0.75) - 1
	idx4 := int(float64(length)*0.8) - 1
	idx5 := int(float64(length)*0.9) - 1
	idx6 := int(float64(length)*0.95) - 1
	idx7 := int(float64(length)*0.98) - 1
	idx8 := int(float64(length)*0.99) - 1
	idx9 := length*1 - 1
	fmt.Println("Thread count: ", threadCount)
	fmt.Println("Each thread request: ", eachThreadRequest)
	fmt.Println("Total request: ", int(threadCount) * eachThreadRequest)
	fmt.Println("Fail: ", countObject.fail)
	fmt.Println("Success: ", countObject.success)
	fmt.Println("Time object count: ", len(timeObject.timeSlice))
	fmt.Println("Max time cost: ", timeObject.timeSlice[length-1])
	fmt.Println("Min time cost: ", timeObject.timeSlice[0])
	fmt.Println("Total time: ", total)
	fmt.Println("Average time cost: ", total / time.Duration(len(timeObject.timeSlice)))
	fmt.Println("50%  ", timeObject.timeSlice[idx1])
	fmt.Println("66%  ", timeObject.timeSlice[idx2])
	fmt.Println("75%  ", timeObject.timeSlice[idx3])
	fmt.Println("80%  ", timeObject.timeSlice[idx4])
	fmt.Println("90%  ", timeObject.timeSlice[idx5])
	fmt.Println("95%  ", timeObject.timeSlice[idx6])
	fmt.Println("98%  ", timeObject.timeSlice[idx7])
	fmt.Println("99%  ", timeObject.timeSlice[idx8])
	fmt.Println("100%  ", timeObject.timeSlice[idx9])
}

func startEachThread(rpcClient grpcpb.ApiServiceClient) {
	defer wg.Done()

	for i:=0;i<eachThreadRequest;i++ {
		start := time.Now()
		req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: name}}
		_, err := rpcClient.GetAccountByName(context.Background(), req)
		end := time.Now()
		if err != nil {
			countObject.addFail()
		} else {
			countObject.addSuccess()
			timeObject.addTimeInterval(end.Sub(start))
		}
	}
}