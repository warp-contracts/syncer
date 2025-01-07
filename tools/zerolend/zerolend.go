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
	"gorm.io/gorm/utils"
)

func main() {
	runBlock(big.NewInt(24512091))
}

func runBlock(number *big.Int) {
	fmt.Println("====== BLOCK ", number)
	client, err := ethclient.Dial(fmt.Sprintf("https://base-mainnet.g.alchemy.com/v2/%s", ""))
	if err != nil {
		log.Fatal("cannot dial", err)
	}

	logNames := make(map[string]string)
	logNames["depositETH"] = "Supply"
	logNames["withdraw"] = "Withdraw"

	asd, p, eee := client.TransactionByHash(context.Background(), common.HexToHash("0x930ac8220a3416e5f6aab5459e64dba419c6996da12fdbae70b85c8e2359357e"))
	fmt.Println("TransactionByHash ", asd.Hash().Hex(), p, eee)

	head, irr := client.HeaderByNumber(context.Background(), number)
	if irr != nil {
		log.Fatal("cannot into head: ", err)
	} else {
		log.Println("fetched block hash ", head.Hash().Hex())
	}

	block, errBlock := client.BlockByNumber(context.Background(), number)
	err = errBlock
	if err != nil {
		log.Fatal("cannot into block: ", err)
	}
	fmt.Println(len(block.Transactions()))
	// txCount, err := client.TransactionCount(context.Background(), head.Hash())
	fmt.Println("Tx count ", len(block.Transactions()))
	// fmt.Println("Tx count", txCount)
	for _, tx := range block.Transactions() {

		// tx, err := client.TransactionInBlock(context.Background(), head.Hash(), uint(i))
		// if err != nil {
		// 	continue
		// }

		if utils.Contains(
			[]string{
				"0xd62899a3c1f12185bc8bfd63411518e1d58ce9548ceba02fd25ebf8c15582984",
			},
			tx.Hash().Hex()) {

			fmt.Println("===================================================================")
			fmt.Println("Tx to", tx.Hash().Hex(), tx.To(), tx.Value(), tx.ChainId(), tx.Data())
			contractAbi, err := eth.GetContractABI(
				"0x80102a3cbAcADa39560555340e1bC567B83C3A80",
				"",
				eth.Base)
			// 		cabi, err := eth.GetContractABIFromFile("InitializableImmutableAdminUpgradeabilityProxy.json")
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// 		fmt.Println("GetContractProxyABI")

			// 		contractAbi, err := eth.GetContractABI(
			// 			tx.To().String(),
			// 			"8QG29V3DJNCAST9APZDWXINNFBWMVHN3AX",
			// 			eth.Sei)

			if err != nil {
				log.Fatal("cannot into contract abi: ", err)
			}
			method, inputsMap, err := eth.DecodeTransactionInputData(contractAbi, tx.Data())
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("method", method)
			fmt.Println("inputsMap: ", inputsMap)

			assets := inputsMap["amount"]
			//referralCode := inputsMap["referralCode"]
			fmt.Println(inputsMap["asset"])
			tokenName := eth.GetTokenName(fmt.Sprintf("%v", inputsMap["asset"]))
			if tokenName == "" {
				tokenName = eth.GetTokenName(tx.To().String())
			}

			fmt.Println("method ", method.RawName, method.String())
			fmt.Println("assets ", assets)
			// 		fmt.Println("referralCode ", referralCode)
			fmt.Println("tokenName ", tokenName)

			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("rawname", method.RawName)
			logName := logNames[method.RawName]
			transferLog, err := GetTransactionLog(receipt, contractAbi, logName, "")
			if err != nil {
				log.Println("FAILURE", logName, err, " <<<< =====================    FAILURE!")
			}
			// transferValue := transferLog
			fmt.Println(transferLog)
			fmt.Println("===================================================================")
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
			fmt.Println("-- Failed", i, err)
			continue
		}

		fmt.Println("event0", event.Name, event.Inputs)
		fmt.Println(event.Name)
		fmt.Println(log)
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
