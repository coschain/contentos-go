package netserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/coschain/contentos-go/p2p/common"
)

// createListener creates a net listener on the port
func createListener(port uint32, isTls bool, CertPath, KeyPath, CAPath string) (net.Listener, error) {
	var listener net.Listener
	var err error

	if isTls {
		listener, err = initTlsListen(port, CertPath, KeyPath, CAPath)
		if err != nil {
			return nil, errors.New("[p2p] initTlslisten failed")
		}
	} else {
		listener, err = initNonTlsListen(port)
		if err != nil {
			return nil, errors.New("[p2p] initNonTlsListen failed")
		}
	}
	return listener, nil
}

//nonTLSDial return net.Conn with nonTls
func nonTLSDial(addr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second*common.DIAL_TIMEOUT)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

//TLSDial return net.Conn with TLS
func TLSDial(nodeAddr, CertPath, KeyPath, CAPath string) (net.Conn, error) {
	clientCertPool := x509.NewCertPool()

	cacert, err := ioutil.ReadFile(CAPath)
	if err != nil {
		return nil, errors.New( fmt.Sprintf("[p2p] load CA file fail", err) )
	}
	cert, err := tls.LoadX509KeyPair(CertPath, KeyPath)
	if err != nil {
		return nil, err
	}

	ret := clientCertPool.AppendCertsFromPEM(cacert)
	if !ret {
		return nil, errors.New("[p2p]failed to parse root certificate")
	}

	conf := &tls.Config{
		RootCAs:      clientCertPool,
		Certificates: []tls.Certificate{cert},
	}

	var dialer net.Dialer
	dialer.Timeout = time.Second * common.DIAL_TIMEOUT
	conn, err := tls.DialWithDialer(&dialer, "tcp", nodeAddr, conf)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

//initNonTlsListen return net.Listener with nonTls mode
func initNonTlsListen(port uint32) (net.Listener, error) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		return nil, errors.New( fmt.Sprintf("[p2p] Error listening %s", err) )
	}
	return listener, nil
}

//initTlsListen return net.Listener with Tls mode
func initTlsListen(port uint32, CertPath, KeyPath, CAPath string) (net.Listener, error) {
	// load cert
	cert, err := tls.LoadX509KeyPair(CertPath, KeyPath)
	if err != nil {
		return nil, errors.New( fmt.Sprintf("[p2p] load keys fail", err) )
	}
	// load root ca
	caData, err := ioutil.ReadFile(CAPath)
	if err != nil {
		return nil, errors.New( fmt.Sprintf("[p2p] read ca fail", err) )
	}
	pool := x509.NewCertPool()
	ret := pool.AppendCertsFromPEM(caData)
	if !ret {
		return nil, errors.New("[p2p]failed to parse root certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}

	listener, err := tls.Listen("tcp", ":"+strconv.Itoa(int(port)), tlsConfig)
	if err != nil {
		return nil, errors.New( fmt.Sprintf("[p2p] tls listen error ", err) )
	}
	return listener, nil
}
