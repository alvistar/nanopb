package usclient

import (
	"github.com/alvistar/nanopb/pkg/nanoipc"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
)

type IUSClient interface {
	Init(conf *ConfNode, l *log.Logger)
	Get(request []byte) ([]byte, error)
}

type USClient struct {
	// Session pool
	sessions      []*nanoipc.Session
	nextSession   int32
	conf          *ConfNode
	needReconnect bool
}

type ConfNode struct {
	Connection string `json:"connection"`
	PoolSize   int    `json:"poolsize"`
}

var logger *log.Entry

func (client *USClient) Init(conf *ConfNode, l *log.Logger) {
	nanoipc.Init(l)

	if l == nil {
		l = log.New()
	}

	logger = l.WithFields(log.Fields{"component": "us_client"})

	if conf == nil {
		client.conf = &ConfNode{
			Connection: "local:///tmp/nano",
			PoolSize:   3,
		}
	} else {
		client.conf = conf
	}

	_ = client.tryConnectNode()
}

// Try connecting to the Nano node
func (client *USClient) tryConnectNode() *nanoipc.Error {
	var err *nanoipc.Error
	client.sessions = make([]*nanoipc.Session, 0)
	for i := 0; err == nil && i < client.conf.PoolSize; i++ {
		session := &nanoipc.Session{}
		client.sessions = append(client.sessions, session)
		err = session.Connect(client.conf.Connection)
	}
	if len(client.sessions) < client.conf.PoolSize {
		logger.Info("Reconnection attempt required")
		client.needReconnect = true
	} else {
		client.needReconnect = false
	}

	if err != nil {
		logger.Error(err.Message)
	}

	return err
}

func (client *USClient) reconnectNode() *nanoipc.Error {
	for _, element := range client.sessions {
		_ = element.Close()
	}
	client.sessions = nil
	client.nextSession = 0
	return client.tryConnectNode()
}

// getSession returns the next available session in a round-robin fashion
func (client *USClient) getSession() *nanoipc.Session {
	next := atomic.AddInt32(&client.nextSession, 1)
	next = next % int32(client.conf.PoolSize)
	return client.sessions[next]
}

func (client *USClient) Get(request []byte) ([]byte, error) {
	var err *nanoipc.Error
	var reply []byte

	if client.needReconnect {
		if err = client.tryConnectNode(); err == nil {
			logger.Info("Reconnected successfully to node")
		}
	}

	reply, err = client.getSession().Request(string(request))

	if err!=nil && err.Category == "Network" {
		if err = client.reconnectNode(); err != nil {
			logger.Error("Unable to reconnect to node")
		}
	}

	if err == nil {
		return reply, nil
	} else {
		return reply, err
	}
}
