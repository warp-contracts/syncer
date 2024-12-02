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
)

func main() {
	//runBlock(big.NewInt(117750972))
	runBlock(big.NewInt(115655361))
	//runBlock(big.NewInt(115655059))
}

func runBlock(number *big.Int) {
	fmt.Println("====== BLOCK ", number)
	//client, err := ethclient.Dial("https://1rpc.io/sei-rpc")
	//client, err := ethclient.Dial("https://evm-rpc.sei-apis.com")
	//client, err := ethclient.Dial("https://1rpc.io/sei-rpc")
	client, err := ethclient.Dial("https://sei-evm-rpc.publicnode.com")
	if err != nil {
		log.Fatal("cannot dial", err)
	}

	logNames := make(map[string]string)
	logNames["depositETH"] = "Supply"
	logNames["withdrawETH"] = "Withdraw"

	asd, p, eee := client.TransactionByHash(context.Background(), common.HexToHash("0x8ac05931de6023d67a2f7abbed20703bd41fbaa689a78bc0736e7adcb8d6d9b4"))
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

	fmt.Println("Tx count ", len(block.Transactions()))
	for _, tx := range block.Transactions() {

		if utils.Contains(
			[]string{
				"0x94236a429477248212c884feb1b0e6df23dee1d4d8c54a9dc09f6b45753b6773",
				"0x8ac05931de6023d67a2f7abbed20703bd41fbaa689a78bc0736e7adcb8d6d9b4",
				"0xf6ce91473d12c11313142369bcba9bdad9375670635a8eef535908a3a00aba05",
			},
			tx.Hash().Hex()) {

			fmt.Println("===================================================================")
			fmt.Println("Tx to", tx.Hash().Hex(), tx.To(), tx.Value(), tx.ChainId(), tx.Data())

			cabi, err := eth.GetContractABIFromFile("InitializableImmutableAdminUpgradeabilityProxy.json")
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("GetContractProxyABI")

			contractAbi, err := eth.GetContractABI(
				tx.To().String(),
				"8QG29V3DJNCAST9APZDWXINNFBWMVHN3AX",
				eth.Sei)

			if err != nil {
				log.Fatal("cannot into contract abi: ", err)
			}

			method, inputsMap, err := eth.DecodeTransactionInputData(contractAbi, tx.Data())
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("inputsMap: ", inputsMap)

			assets := inputsMap["amount"]
			referralCode := inputsMap["referralCode"]
			tokenName := eth.GetTokenName(fmt.Sprintf("%v", inputsMap["token"]))
			if tokenName == "" {
				tokenName = eth.GetTokenName(tx.To().String())
			}

			fmt.Println("method ", method.RawName, method.String())
			fmt.Println("assets ", assets)
			fmt.Println("referralCode ", referralCode)
			fmt.Println("tokenName ", tokenName)

			receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
			if err != nil {
				log.Fatal(err)
			}

			logName := logNames[method.RawName]
			transferLog, err := GetTransactionLog(receipt, cabi, logName, "")
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
