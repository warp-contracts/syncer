package eth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
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
	LayerBank Protocol = iota
	Pendle    Protocol = iota
	Venus     Protocol = iota
	ListaDAO  Protocol = iota
)

type Chain int

const (
	Avax     Chain = iota
	Arbitrum Chain = iota
	Mode     Chain = iota
	Manta    Chain = iota
	Bsc      Chain = iota
)

func (chain Chain) RpcProviderUrl() (rpcProviderUrl string, err error) {
	switch chain {
	case Avax:
		rpcProviderUrl = "https://api.avax.network/ext/bc/C/rpc"
		return
	case Arbitrum:
		rpcProviderUrl = "https://arb1.arbitrum.io/rpc"
		return
	case Mode:
		rpcProviderUrl = "https://mainnet.mode.network"
		return
	case Manta:
		rpcProviderUrl = "https://pacific-rpc.manta.network/http"
		return
	case Bsc:
		rpcProviderUrl = "https://bsc-rpc.publicnode.com"
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
	case Mode:
		apiUrl = "https://explorer.mode.network/api"
		return
	case Manta:
		apiUrl = "https://pacific-explorer.manta.network/api"
		return
	case Bsc:
		apiUrl = "https://api.bscscan.com/api"
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
	case LayerBank:
		return "layer_bank"
	case Pendle:
		return "pendle"
	case Venus:
		return "venus"
	case ListaDAO:
		return "lista_dao"
	}
	return ""
}

func (protocol Protocol) GetAbi() string {
	switch protocol {
	case Sommelier, LayerBank, Venus:
		return "direct"
	case ListaDAO:
		return "proxy"
	case Pendle:
		return "IPActionSwapPTV3.json"
	}
	return ""
}

func (chain Chain) String() string {
	switch chain {
	case Avax:
		return "avax"
	case Arbitrum:
		return "arbitrum"
	case Mode:
		return "mode"
	case Manta:
		return "manta"
	case Bsc:
		return "bsc"
	}
	return ""
}

func (chain Chain) Decimals() float64 {
	switch chain {
	case Bsc:
		return 18
	}

	// this is the base decimals value according to ERC-20 spec
	return 18
}

func GetTokenName(contract string) string {
	switch contract {
	case "0xa835F890Fcde7679e7F7711aBfd515d2A267Ed0B":
		return "binancecoin"
	case "0xb0b84d294e0c75a6abe60171b70edeb2efd14a1b":
		return "binancecoin"
	case "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c":
		return "bitcoin"
	case "0xB68443Ee3e828baD1526b3e0Bdf2Dfc6b1975ec4":
		return "binancecoin"
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

func GetContractProxyABI(contractAddress, apiKey string, chain Chain) (*abi.ABI, error) {
	abiProxies := map[string]string{
		"0xB68443Ee3e828baD1526b3e0Bdf2Dfc6b1975ec4": "0x3a0f552C0555468A9f8Ab641FE44F5ba86208A9C",
		"0xa835F890Fcde7679e7F7711aBfd515d2A267Ed0B": "0xF85D7C7BaF867A97A91fEB9583464B9D44D40a99",
	}
	return GetContractABI(abiProxies[contractAddress], apiKey, chain)
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
		return
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
	ether, _ := new(big.Float).Quo(new(big.Float).SetInt(wei), big.NewFloat(params.Ether)).Float64()
	return ether
}

func GetContractABIFromFile(fileName string) (*abi.ABI, error) {
	pwd, _ := os.Getwd()

	fileData, err := os.Open(fmt.Sprintf("%s/src/warpy_sync/files/%s", pwd, fileName))

	if err != nil {
		return nil, err
	}

	byteValue, err := io.ReadAll(fileData)
	if err != nil {
		return nil, err
	}

	rawABIResponse := &RawABIResponse{}

	err = json.Unmarshal(byteValue, rawABIResponse)
	if err != nil {
		return nil, err
	}

	fileData.Close()

	contractABI, err := abi.JSON(strings.NewReader(*rawABIResponse.Result))
	if err != nil {
		return nil, err
	}
	return &contractABI, nil
}

func GetTransactionLog(receipt *types.Receipt, contractABI *abi.ABI, name string) (eventMap map[string]interface{}, err error) {
	for _, vLog := range receipt.Logs {
		event, err := contractABI.EventByID(vLog.Topics[0])
		if err != nil {
			fmt.Println()
			continue
		}

		if event.Name == name {
			eventMap := make(map[string]interface{})
			eventMap["name"] = event.Name

			indexed := make([]abi.Argument, 0)
			for _, input := range event.Inputs {
				if input.Indexed {
					indexed = append(indexed, input)
				}
			}
			err := abi.ParseTopicsIntoMap(eventMap, indexed, vLog.Topics[1:])

			if err != nil {
				return nil, err
			}

			if len(vLog.Data) > 0 {
				err = contractABI.UnpackIntoMap(eventMap, event.Name, vLog.Data)
				if err != nil {
					return nil, err
				}
			}
			return eventMap, nil
		}
	}

	err = errors.New("desired transaction log not found")
	return
}

func GetPriceInEth(id string) (ethPrice float64, err error) {
	type EthPrice map[string]map[string]float64

	httpClient := resty.New()
	resp, err := httpClient.SetBaseURL("https://api.coingecko.com").R().
		SetResult(EthPrice{}).
		ForceContentType("application/json").
		SetQueryParams(map[string]string{
			"ids":           id,
			"vs_currencies": "eth",
		}).
		SetHeader("Accept", "application/json").
		Get("/api/v3/simple/price")

	if err != nil {
		return
	}

	coinPayload := resp.Result().(*EthPrice)

	if prices, ok := (*coinPayload)[id]; ok {
		if ethPrice, ok = prices["eth"]; ok {
			return
		} else {
			err = fmt.Errorf("no price data for %s in ETH", id)
			return
		}
	} else {
		err = fmt.Errorf("no data for %s", id)
		return
	}
}
