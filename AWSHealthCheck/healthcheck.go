package AWSHealthCheck

import (
	"context"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const (
	MaxDelayTime = 300  // second
)

var HealthCheckName = "healthcheck"

type AWSHealthCheck struct{
	srv *http.Server
	ctx *node.ServiceContext
	log  *logrus.Logger
	healthCheckON   bool
}

func NewAWSHealthCheck(ctx *node.ServiceContext, lg *logrus.Logger) (*AWSHealthCheck, error) {
	return &AWSHealthCheck{log:lg, ctx:ctx, healthCheckON:false}, nil
}

func (this *AWSHealthCheck) Start(node *node.Node) error {
	http.HandleFunc("/", this.myHandler)

	this.startServer(this.ctx)

	s, err := this.ctx.Service(iservices.ConsensusServerName)
	if err != nil {
		return err
	}
	ctrl := s.(iservices.IConsensus)

	go func(){
		for {
			headBlock, err := getHeadBlock(ctrl)
			if headBlock == nil || err != nil {
				this.log.Error("HealthCheck can not fetch head block ", err)
				time.Sleep(time.Second * 10)
				continue
			}

			lastCommitBlock, err := getLastCommitBlock(ctrl)
			if lastCommitBlock == nil || err != nil {
				this.log.Error("HealthCheck can not fetch last commit block ", err)
				time.Sleep(time.Second * 10)
				continue
			}

			now := uint64(time.Now().Unix())
			timeBetweenHeadBlockAndNow := now - headBlock.Timestamp()
			timeBetweenLIBAndHeadBlock := headBlock.Timestamp() - lastCommitBlock.Timestamp()

			if ctrl.CheckSyncFinished() && timeBetweenHeadBlockAndNow <= MaxDelayTime && timeBetweenLIBAndHeadBlock <= MaxDelayTime {
				if !this.healthCheckON {
					this.startServer(this.ctx)
				}
			} else {
				if this.healthCheckON {
					this.stopServer()
				}
			}

			time.Sleep(time.Second)
		}
	}()
	return nil
}

func (this *AWSHealthCheck) Stop() error {
	this.stopServer()
	return nil
}

func (this *AWSHealthCheck) myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "——hi aws ALB, I'm alive ——\n")
}

func (this *AWSHealthCheck) startServer(ctx *node.ServiceContext) {
	this.srv = &http.Server{Addr: fmt.Sprintf(":%s", ctx.Config().HealthCheck.Port)}
	go this.srv.ListenAndServe()
	this.healthCheckON = true
}

func (this *AWSHealthCheck) stopServer() {
	ctx, _ := context.WithTimeout(context.Background(), 5 * time.Second)
	if err := this.srv.Shutdown(ctx); err != nil {
		this.log.Error("HealthCheck server shutdown error ", err)
	}
	this.healthCheckON = false
}

func getHeadBlock(ctrl iservices.IConsensus) (common.ISignedBlock, error) {
	headBlockId := ctrl.GetHeadBlockId()

	if headBlockId.BlockNum() == 0 {
		return nil , errors.New("Chain empty")
	}

	iSignedBlock, err := ctrl.FetchBlock(headBlockId)

	return iSignedBlock, err
}

func getLastCommitBlock(ctrl iservices.IConsensus) (common.ISignedBlock, error) {
	lastCommitBlockId := ctrl.GetLIB()

	if lastCommitBlockId.BlockNum() == 0 {
		return nil , errors.New("Consensus not start up")
	}

	iSignedBlock, err := ctrl.FetchBlock(lastCommitBlockId)

	return iSignedBlock, err
}