package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/warp-contracts/syncer/src/utils/eth"
)

func main() {
	client, err := ethclient.Dial("https://arb1.arbitrum.io/rpc")
	if err != nil {
		log.Fatal(err)
	}

	header, err := client.HeaderByNumber(context.Background(), big.NewInt(223091335))
	if err != nil {
		log.Fatal(err)
	}

	txCount, err := client.TransactionCount(context.Background(), header.Hash())
	if err != nil {
		log.Fatal("tx count not executed")
	}

	for i := 0; i < int(txCount); i++ {
		tx, err := client.TransactionInBlock(context.Background(), header.Hash(), uint(i))
		if err != nil {
			continue
		}

		if tx.Hash().Hex() == "0xf05df1f98133fc32a32479eb6a0a85a9a1b697fa8ad348d30900b4d09dee829a" {
			data, err := os.Open("src/warpy_sync/abi/IPActionSwapPTV3.json")

			if err != nil {
				fmt.Println(err)
			}

			byteValue, _ := io.ReadAll(data)

			rawABIResponse := &eth.RawABIResponse{}

			err = json.Unmarshal(byteValue, rawABIResponse)
			if err != nil {
				log.Fatal(err)
			}

			data.Close()

			contractAbi, _ := abi.JSON(strings.NewReader(*rawABIResponse.Result))

			if err != nil {
				log.Fatal(err)
			}

			method, inputsMap, err := eth.DecodeTransactionInputData(&contractAbi, tx.Data())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(method)
			fmt.Println(inputsMap)
		}

	}

}
