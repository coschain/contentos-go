package detector

import "sync"

type NodeInfo struct {
	version      string
}

type NodeSet struct {
	sync.Mutex
	NodeInfoList     map[string]*NodeInfo
}

var QueryList chan string