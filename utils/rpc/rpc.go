package rpc

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"strings"
)

const methodjson = `[{ "type" : "function", "name" : "isValidSignature", "constant" : true, "inputs": [{"name": "hash", "type": "bytes32"}, {"name": "signature", "type": "bytes"}], "outputs": [{"type": "magicValue", "type": "bytes4"}] }]`

func ValidateDapperSignature(address, value, signature string) (bool, error) {

	client, _ := ethclient.Dial("https://mainnet.infura.io/v3/0720b4fd81a94f9db49ddd00257e1b59")
	defer client.Close()

	a, err := abi.JSON(strings.NewReader(methodjson))

	h := crypto.Keccak256([]byte(value))
	hash := [32]byte{}
	copy(hash[:], h)
	addr := common.HexToAddress(address)

	packed, err := a.Pack("isValidSignature", hash, common.FromHex(signature))
	if err != nil {
		fmt.Println(err)
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

	cmp := bytes.Compare(magic, common.FromHex("0x1626ba7e"))
	if cmp == 0 {
		return true, nil
	}

	return false, nil
}
