package main

import (
	"context"
	"github.com/alvistar/gonano/nanoclient/mocks"
	pb "github.com/alvistar/gonano/nanoproto"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func jsonMatch(t *testing.T, expected string) interface{} {
	return mock.MatchedBy(func(x []byte) bool {
		assert.JSONEq(t, expected, string(x))
		return true
	})
}

func TestBasic(t *testing.T) {
	assert.Equal(t, 1, 1)
}

func TestBlocksInfo(t *testing.T) {
	client := mocks.INanoClient{}
	returned := []byte(`{
  "blocks": {
    "87434F8041869A01C8F6F263B87972D7BA443A72E0A97D7A3FD0CCC2358FD6F9": {
      "block_account": "nano_1ipx847tk8o46pwxt5qjdbncjqcbwcc1rrmqnkztrfjy5k7z4imsrata9est",
      "amount": "30000000000000000000000000000000000",
      "balance": "5606157000000000000000000000000000000",
      "height": "58",
      "local_timestamp": "0",
      "confirmed": "true",
      "contents": {
        "type": "state",
        "account": "nano_1ipx847tk8o46pwxt5qjdbncjqcbwcc1rrmqnkztrfjy5k7z4imsrata9est",
        "previous": "CE898C131AAEE25E05362F247760F8A3ACF34A9796A5AE0D9204E86B0637965E",
        "representative": "nano_1stofnrxuz3cai7ze75o174bpm7scwj9jn3nxsn8ntzg784jf1gzn1jjdkou",
        "balance": "5606157000000000000000000000000000000",
        "link": "5D1AA8A45F8736519D707FCB375976A7F9AF795091021D7E9C7548D6F45DD8D5",
        "link_as_account": "nano_1qato4k7z3spc8gq1zyd8xeqfbzsoxwo36a45ozbrxcatut7up8ohyardu1z",
        "signature": "82D41BC16F313E4B2243D14DFFA2FB04679C540C2095FEE7EAE0F2F26880AD56DD48D87A7CC5DD760C5B2D76EE2C205506AA557BF00B60D8DEE312EC7343A501",
        "work": "8a142e07a10996d5"
      },
      "subtype": "send"
    }
  }
}`)
	client.On("Get", mock.Anything).Return(returned, nil)
	var s = Server{client: &client}
	var msg = pb.BlocksInfoRequest{
		Hashes:[]string{"1234"},
	}
	var err error
	var reply *pb.BlocksInfoReply
	reply, err = s.BlocksInfo(context.Background(), &msg)
	expected := `{"action":"blocks_info", "json_block":"true", "hashes": ["1234"]}`
	client.AssertCalled(t, "Get", jsonMatch(t, expected))
	assert.Nil(t, err)
	msh := jsonpb.Marshaler{OrigName: true}
	replys, err := msh.MarshalToString(reply)
	require.Nil(t, err)
	assert.JSONEq(t, string(returned), replys)
	assert.Equal(t, "30000000000000000000000000000000000", reply.Blocks["87434F8041869A01C8F6F263B87972D7BA443A72E0A97D7A3FD0CCC2358FD6F9"].Amount)
}

//func TestJSON (t *testing.T) {
//	j:=[]byte(`{
//    	"count": "25238",
//    	"unchecked": "14465538"
//			}`)
//
//	count, _ := jsonparser.GetString(j, "count")
//	print(count)
//}