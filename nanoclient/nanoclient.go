package nanoclient

import (
	"encoding/json"
	"github.com/alvistar/gonano/pkg/nanoipc"
	"io/ioutil"
	"log"
	"sync/atomic"
)
type INanoClient interface{
	Init()
	Get(request []byte) ([]byte, *nanoipc.Error)
}

type NanoClient struct{
	// Session pool
	sessions      []*nanoipc.Session
	nextsession   int32
	conf          *ConfNode
	needReconnect bool
}

type ConfNode struct {
	Connection string `json:"connection"`
	Poolsize   int    `json:"poolsize"`
}

func (client *NanoClient) Init() {

	client.conf = & ConfNode{
		Connection: "local:///tmp/nano",
		Poolsize: 1,
	}

	if configBytes, err := ioutil.ReadFile("config.json"); err != nil {
		log.Println("No config file found, using defaults")
	} else {
		if err := json.Unmarshal(configBytes, client.conf); err != nil {
			log.Fatal(err)
		}
	}

	_ = client.tryConnectNode()
}

// Try connecting to the Nano node
func (client *NanoClient) tryConnectNode() *nanoipc.Error {
	var err *nanoipc.Error
	client.sessions = make([]*nanoipc.Session, 0)
	for i := 0; err == nil && i < client.conf.Poolsize; i++ {
		session := &nanoipc.Session{}
		client.sessions = append(client.sessions, session)
		err = session.Connect(client.conf.Connection)
	}
	if len(client.sessions) < client.conf.Poolsize {
		log.Println("Reconnection attempt required")
		client.needReconnect = true
	} else {
		client.needReconnect = false
	}

	if err != nil {
		log.Println(err.Message)
	}

	return err
}

func (client *NanoClient) reconnectNode() *nanoipc.Error {
	for _, element := range client.sessions {
		element.Close()
	}
	client.sessions = nil
	client.nextsession = 0
	return client.tryConnectNode()
}

// getSession returns the next available session in a round-robin fashion
func (client *NanoClient) getSession() *nanoipc.Session {
	next := atomic.AddInt32(&client.nextsession, 1)
	next = next % int32(client.conf.Poolsize)
	return client.sessions[next]
}

func (client *NanoClient) Get(request []byte) ([]byte, *nanoipc.Error) {
	return client.getSession().Request(string(request))
}