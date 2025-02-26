package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-resty/resty/v2"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/warpy_sync"
	"gorm.io/gorm/utils"
)

var (
	conf, _ = config.Load("")
)

func main() {
	fmt.Println("====== Start main ")
	llogggo := logger.NewSublogger("with-config")
	client, _ := eth.GetEthClient(llogggo, conf.WarpySyncer.SyncerChain, "")

	contractAbi, err := warpy_sync.ContractAbiFromMap(conf)

	assetsCalculator := warpy_sync.NewAssetsCalculator(conf).
		WithEthClient(client).
		WithContractAbi(contractAbi)

	readBlock(133637679, client, assetsCalculator)
	readBlock(133130885, client, assetsCalculator)
	readBlock(133637681, client, assetsCalculator)
	readBlock(133622049, client, assetsCalculator)

	if err != nil {
		llogggo.Println("FAILURE", err, " <<<< =====================    FAILURE!")
	}
}

func readBlock(number int64, client *ethclient.Client, calc *warpy_sync.AssetsCalculator) {
	fmt.Println("====== BLOCK ", number)

	var transactions types.Transactions
	block, err := client.BlockByNumber(context.Background(), big.NewInt(number))
	if err != nil {
		log.Println("cannot into block: ", err)

		txes, err := getTxes(number)
		if err != nil {
			log.Fatal("cannot into txes: ", err)
		} else {
			log.Println("txes: ", txes)
		}
		transactions = make([]*types.Transaction, 0)
		for _, sTx := range txes {
			txx, _, txErr := client.TransactionByHash(context.Background(), common.HexToHash(sTx))
			if txErr == nil {
				transactions = append(transactions, txx)
			}
		}
	} else {
		transactions = block.Transactions()
	}

	fmt.Println("Tx count ", len(transactions))
	for _, tx := range transactions {

		if utils.Contains(
			[]string{
				"0x7f1b161f5c26d5a65f728b92b24e0e55ebec59a16e41a2b4cde1931209e7ee31", // sei supply
				"0x23bf0b91363e8abfdfd1ebb98080116883280181a504a9b37e1159712beb228a", // sei supply
				"0x224ede1539ac91769c35a86c29ce7f1f698a6fd88ead14488226eaf305dfa09b", // sei witdraw
				"0x1cce830da6ea2a42564aa64415841ed2e3225fc3512aeb8675f44a3784261d9b", // isei Withdraw
			},
			tx.Hash().Hex()) {

			fmt.Println("Tx ", tx.Hash().Hex(), tx.To(), tx.Value(), tx.ChainId())
			fmt.Println("GetContractProxyABI")

			contractAbi, err := eth.GetContractProxyABI(
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

			//assets := inputsMap["amount"]
			referralCode := inputsMap["referralCode"]
			tokenName := eth.GetTokenName(fmt.Sprintf("%v", inputsMap["token"]))
			if tokenName == "" {
				tokenName = eth.GetTokenName(tx.To().String())
			}

			fmt.Println("tokenName ", tokenName)
			fmt.Println("method ", method.RawName, method.String())
			fmt.Println("referralCode ", referralCode)

			assetsNames := calc.GetAssetsNames(method.RawName, warpy_sync.FromInput)
			fmt.Println("assetsNames ", assetsNames)

			assets, err := calc.GetAssetsFromLog(method.RawName, tx, assetsNames)
			if err != nil {
				log.Println("FAILURE", method.RawName, err, " <<<< =====================    FAILURE!")
			} else {
				log.Println("Found assets: ", assets)
				fmt.Println("===================================================================")
			}
		}

	}

}

// https://seitrace.com/pacific-1/gateway/api/v1/blocks/115655361/transactions?type=EVM

func getTxes(blockNumber int64) (txes []string, err error) {
	type Txes map[string][]map[string]any
	txes = make([]string, 0)

	httpClient := resty.New()
	resp, err := httpClient.SetBaseURL("https://seitrace.com").R().
		SetResult(Txes{}).
		ForceContentType("application/json").
		SetQueryParams(map[string]string{
			"type": "EVM",
		}).
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("/pacific-1/gateway/api/v1/blocks/%d/transactions", blockNumber))

	if err != nil {
		return
	}

	coinPayload := resp.Result().(*Txes)

	if items, ok := (*coinPayload)["items"]; ok {
		for _, item := range items {
			txes = append(txes, (item["hash"]).(string))
		}
	} else {
		err = fmt.Errorf("no data for %d", blockNumber)
		return
	}
	return
}
