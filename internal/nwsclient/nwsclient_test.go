package nwsclient

import (
	pb "github.com/alvistar/nanopb/nanoproto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var MESSAGE = `
{
  "topic": "confirmation",
  "time": "1564935350664",
  "message": {
    "account": "nano_1tgkjkq9r96zd3pkr7edj8e4qbu3wr3ps6ettzse8hmoa37nurua7faupjhc",
    "amount": "15621963968634827029081574961",
    "hash": "0E889F83E28152A70E87B92D846CA3D8966F3AEEC65E11B25F7B4E6760C57CA3",
    "confirmation_type": "active_quorum",
    "election_info": {
      "duration": "546",
      "time": "1564935348219",
      "tally": "42535295865117307936387010521258262528",
      "request_count": "1" 
    },
    "block": {
      "type": "state",
      "account": "nano_1tgkjkq9r96zd3pkr7edj8e4qbu3wr3ps6ettzse8hmoa37nurua7faupjhc",
      "previous": "4E9003ABD469D1F58A70518234016797FA654B494A2627B8583052629A91689E",
      "representative": "nano_3rw4un6ys57hrb39sy1qx8qy5wukst1iiponztrz9qiz6qqa55kxzx4491or",
      "balance": "0",
      "link": "3098F4C0D1D8BD889AF078CDFF81E982B8EFA6D6D8FAE954CF0CDC7A256C3F8B",
      "link_as_account": "nano_1e6rym1f5p7xj4fh1y8fzy1ym1orxymffp9tx7cey58whakprhwdzuk533th",
      "signature": "D5C332587B1A4DEA35B6F03B0A9BEB45C5BBE582060B0252C313CF411F72478721F8E7DA83A779BA5006D571266F32BDE34C1447247F417F8F12101D3ADAF705",
      "work": "c950fc037d61e372",
      "subtype": "send"
    }
  }
}
`

func TestSubscription(t *testing.T) {
	client := WSClient{}

	mychan := make(chan pb.SubscriptionEntry)

	client.Subscribe(&mychan, []string {"nano_1e6rym1f5p7xj4fh1y8fzy1ym1orxymffp9tx7cey58whakprhwdzuk533th"})

	go client.subHandler(MESSAGE)

	select {
		case entry := <-mychan:
			assert.Equal(t, "c950fc037d61e372",  entry.Message.Block.Work)
		case <-time.After(3 * time.Second):
			assert.Fail(t, "Timeout")
	}
}

func TestSubscriptionAll(t *testing.T) {
	client := WSClient{}

	mychan := make(chan pb.SubscriptionEntry)

	client.Subscribe(&mychan, []string {})

	go client.subHandler(MESSAGE)

	select {
	case entry := <-mychan:
		assert.Equal(t, "c950fc037d61e372",  entry.Message.Block.Work)
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Timeout")
	}
}