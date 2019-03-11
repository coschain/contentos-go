package myhttp

import (
	"fmt"
	"github.com/coschain/contentos-go/node"
	"github.com/sirupsen/logrus"
	"net/http"
	"context"
)

var HealthCheckName = "healthcheck"

type myhttp struct{
	srv *http.Server
	log  *logrus.Logger
}

func NewMyHttp(ctx *node.ServiceContext, lg *logrus.Logger) (*myhttp, error) {
	s := &http.Server{Addr: ":9090"}
	return &myhttp{srv:s,log:lg}, nil
}

func (this *myhttp) Start(node *node.Node) error {
	http.HandleFunc("/", myHandler)
	go func(){
		if err := this.srv.ListenAndServe(); err != http.ErrServerClosed {
			this.log.Fatalf("ListenAndServe(): %s", err)
		}
	}()
	return nil
}

func (this *myhttp) Stop() error {
	this.srv.Shutdown(context.TODO())
	return nil
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "——hi aws ALB, I'm alive ——\n")
}