package detector

import (
	"fmt"
	"context"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"google.golang.org/grpc"
	"strings"
	"sync"
)

var Wg = &sync.WaitGroup{}

const (
	querylistLength = 512
)

func Init() *NodeSet {
	nodeSet := new(NodeSet)
	nodeSet.NodeInfoList = make(map[string]*NodeInfo)

	QueryList = make(chan string, querylistLength)

	return nodeSet
}

func (manager *NodeSet) AddToQuerylist(list []string) {
	manager.Lock()
	defer manager.Unlock()

	for i:=0;i<len(list);i++ {
		if list[i] != "" {
			_, ok := manager.NodeInfoList[list[i]]
			if !ok {
				QueryList <- list[i]
			}
		}
	}
}

func (manager *NodeSet) restoreInfo (endPoint string, info *NodeInfo) {
	manager.Lock()
	defer manager.Unlock()

	_, ok := manager.NodeInfoList[endPoint]
	if !ok {
		fmt.Printf("Endpoint: %s, Node version: %s\n", endPoint, info.version)
		manager.NodeInfoList[endPoint] = info
	}
}

func parseIP (peerList []string) []string {
	var ips []string

	for i:=0;i<len(peerList);i++ {
		if peerList[i] != "" {
			ipPort := strings.Split(peerList[i], ":")
			ips = append(ips, ipPort[0])
		}
	}

	return ips
}

// Query has two things to do
// 1. query the neighbours of this node and restore them
// 2. query this node's version
func (manager *NodeSet) Query(endPoint string) {
	defer Wg.Done()
	
	var conn *grpc.ClientConn
	conn, err := rpc.Dial(endPoint)

	if err == nil && conn != nil {
		api := grpcpb.NewApiServiceClient(conn)

		neighbourResp, err := api.GetNodeNeighbours(context.Background(), &grpcpb.NonParamsRequest{})
		if err == nil {
			peerList := strings.Split(neighbourResp.Peerlist, ", ")
			iplist := parseIP(peerList)
			manager.AddToQuerylist(iplist)
		}

		versionResp, err := api.GetNodeRunningVersion(context.Background(), &grpcpb.NonParamsRequest{})

		if err == nil {
			info := &NodeInfo{version:versionResp.NodeVersion}
			manager.restoreInfo(endPoint, info)
		}
	}
}