package eth

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type (
	RawABIResponse struct {
		Status  *string `json:"status"`
		Message *string `json:"message"`
		Result  *string `json:"result"`
	}
)

type Protocol int

const (
	Delta     Protocol = iota
	Sommelier Protocol = iota
)

type Chain int

const (
	Avax     Chain = iota
	Arbitrum Chain = iota
)

func (chain Chain) RpcProviderUrl() (rpcProviderUrl string, err error) {
	switch chain {
	case Avax:
		rpcProviderUrl = "https://api.avax.network/ext/bc/C/rpc"
		return
	case Arbitrum:
		rpcProviderUrl = "https://arb1.arbitrum.io/rpc"
		return
	}

	err = errors.New("ETH chain unknown")
	return
}

func (chain Chain) Api() (apiUrl string, err error) {
	switch chain {
	case Arbitrum:
		apiUrl = "https://api.arbiscan.io/api"
		return
	}

	err = errors.New("ETH chain unknown")
	return
}

func (protocol Protocol) String() string {
	switch protocol {
	case Delta:
		return "delta"
	case Sommelier:
		return "sommelier"
	}
	return ""
}

func GetEthClient(log *logrus.Entry, chain Chain) (client *ethclient.Client, err error) {
	rpcProviderUrl, err := chain.RpcProviderUrl()
	if err != nil {
		log.WithError(err).Error("ETH chain unknown")
		return
	}

	client, err = ethclient.Dial(rpcProviderUrl)
	if err != nil {
		log.WithError(err).Error("Cannot get ETH client")
		return
	}

	return
}

func GetContractRawABI(address string, apiKey string, chain Chain) (rawABIResponse *RawABIResponse, err error) {
	apiUrl, err := chain.Api()
	if err != nil {
		return nil, err
	}
	client := resty.New()
	rawABIResponse = &RawABIResponse{}
	resp, err := client.R().
		SetQueryParams(map[string]string{
			"module":  "contract",
			"action":  "getabi",
			"address": address,
			"apikey":  apiKey,
		}).
		SetResult(rawABIResponse).
		Get(apiUrl)

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf(fmt.Sprintf("Get contract raw abi was not successful: %s\n", resp))
	}

	if *rawABIResponse.Status != "1" {
		return nil, fmt.Errorf(fmt.Sprintf("Get contract raw abi failed: %s\n", *rawABIResponse.Result))
	}

	return rawABIResponse, nil
}

func GetContractABI(contractAddress, apiKey string, chain Chain) (*abi.ABI, error) {
	rawABIResponse, err := GetContractRawABI(contractAddress, apiKey, chain)
	if err != nil {
		return nil, err
	}

	contractABI, err := abi.JSON(strings.NewReader(*rawABIResponse.Result))
	if err != nil {
		return nil, err
	}
	return &contractABI, nil
}

func DecodeTransactionInputData(contractABI *abi.ABI, data []byte) (method *abi.Method, inputsMap map[string]interface{}, err error) {
	if len(data) == 0 {
		err = errors.New("no data to decode")
		return
	}
	methodSigData := data[:4]
	inputsSigData := data[4:]
	method, err = contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatal(err)
	}
	inputsMap = make(map[string]interface{})
	err = method.Inputs.UnpackIntoMap(inputsMap, inputsSigData)
	return
}

func GetTxSenderHash(tx *types.Transaction) (txSenderHash string, err error) {
	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	txSenderHash = sender.String()
	return
}

func WeiToEther(wei *big.Int) float64 {
	return float64(wei.Int64()) / params.Ether
}
