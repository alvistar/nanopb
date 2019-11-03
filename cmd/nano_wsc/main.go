package main

import (
	"github.com/alvistar/nanopb/internal/nwsclient"
	pb "github.com/alvistar/nanopb/nanoproto"
	"log"
	"os"
	"os/signal"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	client := nwsclient.WSClient{}
	client.Init()

	mychan := make(chan pb.SubscriptionEntry)

	//s := Subscription{
	//	channel:  &mychan,
	//	accounts: []string {"nano_1jrd1ri7dfo1gyh9iqqmtfk1aq64oi9c57xixtjdosfjwmxpkebpuruuen34"},
	//}

	client.Subscribe(&mychan, []string {"nano_1jrd1ri7dfo1gyh9iqqmtfk1aq64oi9c57xixtjdosfjwmxpkebpuruuen34"})

	for {
		select {

		case <- client.Done:
			return

		case msg := <-mychan:
			log.Println("channel:", msg)

		case <-interrupt:
			log.Println("interrupt")
			client.Close()
		}
	}
}

