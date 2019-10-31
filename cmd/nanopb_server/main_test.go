package main

import (
	"context"
	"github.com/alvistar/gonano/internal/nanoclient/mocks"
	pb "github.com/alvistar/gonano/nanoproto"
	"github.com/golang/protobuf/jsonpb"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

var token = "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJzb21lIjoicGF5bG9hZCJ9.E-l6kPchs7uJXSoCuS2XcPjhJJlZrcqPfw39AdHS_gp_rLrzESMPU6M5R-TBB9Teb6W0P63pDlYBG0Rm82sblRDQfpCgpPY9E2M2xzISYQHRGcnc6reuviirISzTA3LNSKkJHYw2kSqxtohRFF56DIditTB28TDFRB0dN9T08aCTlZOIrUTBWdlROD0dXdiJ8Spyh1VpQbxOq7rSzaiEmTruiH-JErCtPxXphKI4ZUG48m0aR-K6RMmIhC9bX8KVPMHQLYSckFdUxyFQJU56Rn-OcB3AhiIN_rvTJ3qpRtUAxo-Fe09mobfFyKHxZYdYDlZi_jf6pjOup8AbcCD4Og"
var pubkey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsg5BDY/YFhgoU2xmvOo0
vzauJGUUUPufHXyhZI6fb/a7MTRXAnZBexzVv3V6SyKNBpbskUMleIbYjFWJiARO
k/tnr6smQTWW+pkC2kftdfA3jmBL1gJuqift3M5MARfAOkGT3gsP2Z/coml3kEBl
EU/fspus0xrSNU/T3op6UIQhL80YgW/rvGaDifSFmEevBWA9KZHHU/qYgLea2ETF
mxtlT0SgCIFMiMbHJGjkeQYhUo5tTRvssuZgz8Ks/81YF+GYdzGL4DQhLODF7fc6
TduTckMWs+2b6NMcwlEJCF0NCRiTl9YL4nZJP4hrpnjUaZVEtJ6/Yms8B6AFvzjI
2QIDAQAB
-----END PUBLIC KEY-----`

var pubkeywrong = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsg5BDY/YFhgoU2xmvOo0
vzauJGUUUPufHXyhZI6fb/a7MTRXAnZBexzVv3V6SyKNBpbskUMleIbYjFWJiARO
k/tnr6smQTWW+pkC2kftdfB3jmBL1gJuqift3M5MARfAOkGT3gsP2Z/coml3kEBl
EU/fspus0xrSNU/T3op6UIQhL80YgW/rvGaDifSFmEevBWA9KZHHU/qYgLea2ETF
mxtlT0SgCIFMiMbHJGjkeQYhUo5tTRvssuZgz8Ks/81YF+GYdzGL4DQhLODF7fc6
TduTckMWs+2b6NMcwlEJCF0NCRiTl9YL4nZJP4hrpnjUaZVEtJ6/Yms8B6AFvzjI
2QIDAQAB
-----END PUBLIC KEY-----`

var returned = []byte(`{
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

	func jsonMatch(t *testing.T, expected string) interface{} {
	return mock.MatchedBy(func(x []byte) bool {
		assert.JSONEq(t, expected, string(x))
		return true
	})
}

func TestMain(m *testing.M) {
	logger = log.NewEntry(log.New())
	m.Run()
}

func TestBasic(t *testing.T) {
	assert.Equal(t, 1, 1)
}

func TestBlocksInfo(t *testing.T) {

	client := mocks.INanoClient{}

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

func TestBlocksInfoError(t *testing.T) {

	client := mocks.INanoClient{}

	client.On("Get", mock.Anything).Return([]byte(`{"error":"myerror"}`), nil)
	var s = Server{client: &client}
	var msg = pb.BlocksInfoRequest{
		Hashes:[]string{"1234"},
	}
	var err error
	var reply *pb.BlocksInfoReply
	reply, err = s.BlocksInfo(context.Background(), &msg)
	expected := `{"action":"blocks_info", "json_block":"true", "hashes": ["1234"]}`
	client.AssertCalled(t, "Get", jsonMatch(t, expected))
	assert.Nil(t, reply)
	assert.Error(t, err)
	assert.Equal(t, "myerror", err.Error())

}

func TestGetAction(t *testing.T) {
	request := pb.AccountsBalancesRequest{Accounts: []string {"123"}}
	msg, _ := getAction(&request, "test", nil)

	assert.JSONEq(t, `{"action":"test", "accounts": ["123"]}`, msg)
}

func TestGetActionWithOptions(t *testing.T) {
	request := pb.AccountsBalancesRequest{Accounts: []string {"123"}}
	msg, _ := getAction(&request, "test", map[string]string {"options":"opt1"})

	assert.JSONEq(t, `{"action":"test", "accounts": ["123"], "options":"opt1"}`, msg)
}

//func TestReflection(t *testing.T) {
//	request := pb.AccountsBalancesRequest{Accounts: []string {"123"}}
//	assert.Equal(t, "account_balances", reflect.TypeOf(request).String())
//}


func TestValid(t *testing.T) {
	assert.True(t, valid([]string{token}, []byte(pubkey)))
}

func TestValidWrongKey(t *testing.T) {
	assert.False(t, valid([]string{token}, []byte(pubkeywrong)))

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