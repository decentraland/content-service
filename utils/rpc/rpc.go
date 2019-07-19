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

func ValidateDapperSignature(address, msg, signature string) (bool, error) {

	client, _ := ethclient.Dial("https://mainnet.infura.io/v3/0720b4fd81a94f9db49ddd00257e1b59")
	defer client.Close()

	a, err := abi.JSON(strings.NewReader(methodjson))

	Signature := signature
	Value := msg
	Address := common.HexToAddress(address)
	Hash := [32]byte{}
	copy(Hash[:], common.FromHex(Value)[0:32])
	packed, err := a.Pack("isValidSignature", Hash, common.FromHex(Signature))
	if err != nil {
		return false, err
	}
	callMsg := ethereum.CallMsg{
		To:   &Address,
		Data: packed,
	}

	fmt.Println(common.ToHex(callMsg.Data))
	res, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return false, err
	}

	magic := res[0:4]

	cmp := bytes.Compare(magic, common.FromHex("0x1626ba7e"))
	if cmp == 0 {
		return true, nil
	} else {
		return false, nil
	}
}
