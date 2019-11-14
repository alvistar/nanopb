package nwsclient

import (
	"encoding/json"
	pb "github.com/alvistar/nanopb/nanoproto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/url"
	"sync"
	"time"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

type Subscription struct {
	channel  *chan pb.SubscriptionEntry
	accounts []string
}

type WSClient struct {
	Done          chan struct{}
	LocalAccounts bool
	conn          *websocket.Conn
	subscriptions sync.Map
	logger        *log.Entry
}

func (client *WSClient) wsprocess() {
	client.logger.Debug("Starting wsprocess")
	defer close(client.Done)
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			//if websocket.IsCloseError(err, 1000) {
			//	client.logger.Error("closing error")
			//	return
			//}
			client.logger.Error("read:", err)
			client.logger.Error("connection lost - terminating")
			panic("connection lost with websocket")
		}

		client.subHandler(string(message))
	}
}

func (client *WSClient) subHandler(message string) {
	entry := pb.SubscriptionEntry{}

	if err := jsonpb.UnmarshalString(message, &entry); err != nil {
		client.logger.Error("error unmarshaling message: ", err.Error())
		return
	}

	client.logger.Debugln("received:", entry)

	client.subscriptions.Range(
		func(key, value interface{}) bool {
			subscription := value.(*Subscription)

			if len(subscription.accounts) == 0 ||
				stringInSlice(entry.Message.Block.LinkAsAccount, subscription.accounts) {

				select {
				case *subscription.channel <- entry:
				default:
				}
			}
			return true
		})
}

func (client *WSClient) Subscribe(channel *chan pb.SubscriptionEntry, account []string) {
	s := Subscription{
		channel:  channel,
		accounts: account,
	}

	client.subscriptions.Store(channel, &s)

}

func (client *WSClient) Unsubscribe(channel *chan pb.SubscriptionEntry) {
	client.subscriptions.Delete(channel)
}

func (client *WSClient) Close() {
	client.logger.Info("Closing connection")
	err := client.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		client.logger.Error("write close:", err)
		return
	}
	select {
	case <-client.Done:
	case <-time.After(time.Second):
	}

	_ = client.conn.Close()
}

func (client *WSClient) Init(l *log.Logger) {
	var err error

	client.Done = make(chan struct{})

	u := url.URL{Scheme: "ws", Host: "127.0.0.1:7078", Path: ""}

	if l == nil {
		l = log.New()
	}

	client.logger = l.WithFields(log.Fields{"component": "nwsclient"})
	client.logger.Info("connecting to ", u.String())

	client.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		client.logger.Fatal("dial:", err)
	}

	client.logger.Info("connected")

	// accounts := []string{"nano_1jrd1ri7dfo1gyh9iqqmtfk1aq64oi9c57xixtjdosfjwmxpkebpuruuen34"}

	request := map[string]interface{}{
		"action": "subscribe",
		"topic":  "confirmation",
		"ack":    "false",
		//"options": map[string]interface{} {
		//	"accounts": accounts,
		//},
	}

	if client.LocalAccounts {
		request["options"]= map[string]interface{} {
			"all_local_accounts": true,
		}
	}

	data, _ := json.Marshal(request)

	client.logger.Info("Request: ", string(data))

	err = client.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		log.Fatal("subscribing:", err)
	}

	go client.wsprocess()

}
