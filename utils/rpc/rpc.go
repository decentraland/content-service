package rpc

import (
	"bytes"
	"context"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const methodjson = `[{ "type" : "function", "name" : "isValidSignature", "constant" : true, "inputs": [{"name": "hash", "type": "bytes32"}, {"name": "signature", "type": "bytes"}], "outputs": [{"type": "magicValue", "type": "bytes4"}] }]`

type RPC interface {
	ValidateDapperSignature(address, value, signature string) (bool, error)
}

type rpcClient struct {
	url string
}

func NewRPC(url string) RPC {
	return &rpcClient{url: url}
}

// ERC-1654 uses this constant when a signature is valid
var expected = []byte{0x16, 0x26, 0xba, 0x7e}

func (r *rpcClient) ValidateDapperSignature(address, value, signature string) (bool, error) {

	client, _ := ethclient.Dial(r.url)
	defer client.Close()

	a, err := abi.JSON(strings.NewReader(methodjson))

	h := crypto.Keccak256([]byte(value))
	hash := [32]byte{}
	copy(hash[:], h)
	addr := common.HexToAddress(address)

	packed, err := a.Pack("isValidSignature", hash, common.FromHex(signature))
	if err != nil {
		return false, err
	}

	callMsg := ethereum.CallMsg{
		To:   &addr,
		Data: packed,
	}

	res, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return false, err
	}

	magic := res[0:4]

	return bytes.Compare(magic, expected) == 0, nil
}
