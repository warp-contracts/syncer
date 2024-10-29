package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"gorm.io/gorm/utils"
	"log"
	"math/big"
	"strings"
)

func main() {
	runBlock(big.NewInt(43136354))
	//runBlock(big.NewInt(43225801))
	//runBlock(big.NewInt(43362115))
	//runBlock(big.NewInt(43362133))
	//runBlock(big.NewInt(43362168))
	runBlock(big.NewInt(43542002))
	//runBlock(big.NewInt(43362177))
}

func runBlock(number *big.Int) {
	client, err := ethclient.Dial("https://bsc-rpc.publicnode.com")
	if err != nil {
		log.Fatal(err)
	}

	block, errBlock := client.BlockByNumber(context.Background(), number)
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

		if utils.Contains(
			[]string{
				"0xbceef6285496d2b1a938b7bdcc93277aee6f6f60dbceea43e4e5c2e16a7458ed",
				"0xee77dcdb12fbeaac8bb45b1f21a4b58367935de7c6510f8b7e96c116541ea928",
				"0x87b9bb7dc7b56e6d4675fa9075d92276af8e9305d1e3c148b2c4e1415f2ae295",
				"0x64447cffd0dfead19176e054a061e390e7e6b5176c820767a13d76129f1dcc9e",
				"0x3d4edbf17fbac088d27051c87e095eb25b3e97289b035be825dcce80b7011baa",
				"0x11e1ed9dd6a4690b2e3db3cbad7a9c3112eda0e01bfeea7a0ff6cfde5b6a1db5",
				"0x7b3c278f97af43f93fd783aa57499e6c51d6b99b9518ae7176b03589a1e2cab5",
			},
			tx.Hash().Hex()) {

			fmt.Println("===================================================================")
			fmt.Println("Tx to", tx.Hash().Hex(), tx.To())

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

			fmt.Println("GetContractProxyABI")

			contractAbi, err := eth.GetContractProxyABI(
				tx.To().String(),
				"8QG29V3DJNCAST9APZDWXINNFBWMVHN3AX",
				eth.Bsc)

			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("DecodeTransactionInputData")
			method, inputsMap, err := eth.DecodeTransactionInputData(contractAbi, tx.Data())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(inputsMap)

			assets := inputsMap["dink"]
			methodName := strings.ToUpper(method.RawName[0:1]) + method.RawName[1:]
			tokenName := eth.GetTokenName(fmt.Sprintf("%v", inputsMap["token"]))
			if tokenName == "" {
				tokenName = eth.GetTokenName(tx.To().String())
			}

			fmt.Println("method", methodName, method.String())
			fmt.Println("assets", assets)
			fmt.Println("tokenName", tokenName)

			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				log.Fatal(err)
			}
			transferLog, err := GetTransactionLog(receipt, contractAbi, "Deposit", "")
			if err != nil {
				log.Println("FAILURE", methodName, err, " <<<< =====================    FAILURE!")
			}
			// transferValue := transferLog
			fmt.Println(transferLog)
		}

	}

}

func GetTransactionLog(receipt *types.Receipt, contractABI *abi.ABI, log string, from string) (output map[string]interface{}, err error) {
	fmt.Println("Logs", len(receipt.Logs))
	for i, vLog := range receipt.Logs {
		// 0xf9ffabca9c8276e99321725bcb43fb076a6c66a54b7f21c4e8146d8519b417dc
		fmt.Println("Log", i, vLog.Topics[0])
		event, err := contractABI.EventByID(vLog.Topics[0])
		if err != nil {
			// fmt.Println(err)
			continue
		}

		fmt.Println("event0", event.Name, event.Inputs)

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

			fmt.Println(inputsDataMap)
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
