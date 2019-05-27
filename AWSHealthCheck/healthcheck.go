package AWSHealthCheck

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
	"net/http"
	"context"
	"time"
)

var HealthCheckName = "healthcheck"

type AWSHealthCheck struct{
	srv *http.Server
	ctx *node.ServiceContext
	log  *logrus.Logger
}

func NewAWSHealthCheck(ctx *node.ServiceContext, lg *logrus.Logger) (*AWSHealthCheck, error) {
	s := &http.Server{Addr: fmt.Sprintf(":%s", ctx.Config().HealthCheck.Port)}
	return &AWSHealthCheck{srv:s,log:lg, ctx:ctx}, nil
}

func (this *AWSHealthCheck) Start(node *node.Node) error {
	http.HandleFunc("/", this.myHandler)
	go func(){
		for {
			s, err := this.ctx.Service(iservices.ConsensusServerName)
			if err != nil {
				return
			}
			ctrl := s.(iservices.IConsensus)
			if ctrl.CheckSyncFinished() {
				if err := this.srv.ListenAndServe(); err != http.ErrServerClosed {
					this.log.Fatalf("ListenAndServe(): %s", err)
				}
				break
			}
			time.Sleep(time.Second / 10)
		}
	}()
	return nil
}

func (this *AWSHealthCheck) Stop() error {
	this.srv.Shutdown(context.TODO())
	return nil
}

func (this *AWSHealthCheck) myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "——hi aws ALB, I'm alive ——\n")
}