package rpc

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"io/ioutil"
)

var address = "0x3B21028719a4ACa7EBee35B0157a6F1B0cF0d0c5"
var msg = "0x8f0ae004c75ba965414633cb2328893e5261436d25bb9a0d80778dac39175a5e"
var signature = "0x26942c624f9eba7f19a22d0dddbf2161249cb35c83a3807a82153b2c2b8f14b371efeba58beac447c87483d462ac5b90d1b996cfff84b85df9d4a0afe76b80b91b5d12a48a349c4d7b3e63b03048f95b0a23cdc65be805f0151099bc0b02d3217107c402c16ac00375792ce59827a234d3b6d4023e8ec72073ad078ad3b305c4b61b"
var data = "0x1626ba7ed3720ee927cdd9f51341a8d85ea2515212f9de92226044a8f0668526dec212c60000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008226942c624f9eba7f19a22d0dddbf2161249cb35c83a3807a82153b2c2b8f14b371efeba58beac447c87483d462ac5b90d1b996cfff84b85df9d4a0afe76b80b91b5d12a48a349c4d7b3e63b03048f95b0a23cdc65be805f0151099bc0b02d3217107c402c16ac00375792ce59827a234d3b6d4023e8ec72073ad078ad3b305c4b61b000000000000000000000000000000000000000000000000000000000000"

//I'm using it to check that no endianess fucks the code
func TestEndianess(t *testing.T) {
	var expected = []byte{0x16, 0x26, 0xba, 0x7e}
	var magic = common.FromHex("0x1626ba7e")
	if bytes.Compare(expected, magic) != 0 {
		t.Fail()
	}
}

func TestRPC(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		contains := strings.Contains(string(body), data)
		expected := `{"id":1,"jsonrpc": "2.0","result": "0x1626ba7e"}`
		unexpected := `{"id":1,"jsonrpc": "2.0","result": "0x00000000"}`
		if contains {
			w.Write([]byte(expected))
		} else {
			w.Write([]byte(unexpected))
		}
	}))

	r := NewRPC(server.URL)
	ret, err := r.ValidateDapperSignature(address, msg, signature)
	if !ret || err != nil {
		t.Fail()
	}

	ret, err = r.ValidateDapperSignature(address, "invalid message", signature)
	if ret || err != nil {
		t.Fail()
	}
}
