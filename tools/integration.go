package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/warp-contracts/syncer/src/utils/eth"
)

func main() {
	client, err := ethclient.Dial("https://bsc-rpc.publicnode.com")
	if err != nil {
		log.Fatal(err)
	}

	block, errBlock := client.BlockByNumber(context.Background(), big.NewInt(41298607))
	err = errBlock
	if err != nil {
		log.Fatal(err)
	}
	// blockTxs = block.Transactions()
	// blockHeight = block.Number().Uint64()
	// blockHash = block.Hash().String()
	// blockTime = block.Time()
	// fmt.Println(header.)

	// txCount, err := client.TransactionCount(context.Background(), header.Hash())
	// if err != nil {
	// 	log.Fatal(err)
	// 	log.Fatal("tx count not executed")
	// }f
	// fmt.Println(block.ReceivedAt)
	for _, tx := range block.Transactions() {
		// tx, err := client.TransactionInBlock(context.Background(), header.Hash(), uint(i))
		// if err != nil {
		// 	continue
		// }

		if tx.Hash().Hex() == "0x315a9aefb94007967e936f27595b8a17856e46be7a3e42d81b1a977a5ccde417" {
			// data, err := os.Open("src/warpy_sync/abi/IPActionSwapPTV3.json")

			// if err != nil {
			// 	fmt.Println(err)
			// }

			// byteValue, _ := io.ReadAll(data)

			// rawABIResponse := &eth.RawABIResponse{}

			// err = json.Unmarshal(byteValue, rawABIResponse)
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// data.Close()

			contractAbi, err := eth.GetContractABI(
				"0xA07c5b74C9B40447a954e1466938b865b6BBea36",
				"N8RD68KAJWVJUWXQBGHM13WKT9ZE2NV1CG",
				eth.Bsc)

			if err != nil {
				log.Fatal(err)
			}
			_, inputsMap, err := eth.DecodeTransactionInputData(contractAbi, tx.Data())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(inputsMap)

			assets := inputsMap["mintAmount"]

			fmt.Println("assets", assets)

			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				log.Fatal(err)
			}
			transferLog, err := GetTransactionLog(receipt, contractAbi, "Mint", "")
			if err != nil {
				log.Fatal(err)
			}
			// transferValue := transferLog
			fmt.Println(transferLog)
		}

	}

}

func GetTransactionLog(receipt *types.Receipt, contractABI *abi.ABI, log string, from string) (output map[string]interface{}, err error) {
	for _, vLog := range receipt.Logs {
		// 0xf9ffabca9c8276e99321725bcb43fb076a6c66a54b7f21c4e8146d8519b417dc
		event, err := contractABI.EventByID(vLog.Topics[0])
		if err != nil {
			// fmt.Println(err)
			continue
		}

		if event.Name == log {
			inputsDataMap := make(map[string]interface{})
			inputsDataMap["name"] = event.Name

			indexed := make([]abi.Argument, 0)
			for _, input := range event.Inputs {
				if input.Indexed {
					indexed = append(indexed, input)
				}
			}

			// fmt.Println(indexed)
			// parse topics without event name
			err := abi.ParseTopicsIntoMap(inputsDataMap, indexed, vLog.Topics[1:])

			if from != "" && from != inputsDataMap["from"].(common.Address).String() {
				continue
			}
			if err != nil {
				fmt.Println(err)
			}

			// if inputsDataMap["from"].(common.Address).String() == "0x64937ab314bc1999396De341Aa66897C30008852" {
			if len(vLog.Data) > 0 {
				// outputDataMap := make(map[string]interface{})
				err = contractABI.UnpackIntoMap(inputsDataMap, event.Name, vLog.Data)
				if err != nil {
					return nil, err
				}
				// fmt.Println(in)
				// fmt.Print(value)
				// return value, nil
			}

			fmt.Println(inputsDataMap)
			output = inputsDataMap
			return output, nil
			// }
		}
	}

	err = errors.New("transfer log not found")
	return
}
