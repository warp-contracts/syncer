package warpy_sync

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"slices"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type AssetsCalculator struct {
	*task.Task

	// db          *gorm.DB
	monitor     monitoring.Monitor
	input       chan *SommelierTransactionPayload
	ethClient   *ethclient.Client
	contractAbi map[string]*abi.ABI
	Output      chan *SommelierTransactionPayload
}

func NewAssetsCalculator(config *config.Config) (self *AssetsCalculator) {
	self = new(AssetsCalculator)

	self.Output = make(chan *SommelierTransactionPayload)

	self.Task = task.NewTask(config, "assets_calculator").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.WarpySyncer.SyncerDepositNumWorkers, config.WarpySyncer.SyncerDepositWorkerQueueSize)

	return
}

func (self *AssetsCalculator) WithInputChannel(v chan *SommelierTransactionPayload) *AssetsCalculator {
	self.input = v
	return self
}

func (self *AssetsCalculator) WithMonitor(monitor monitoring.Monitor) *AssetsCalculator {
	self.monitor = monitor
	return self
}

func (self *AssetsCalculator) WithEthClient(ethClient *ethclient.Client) *AssetsCalculator {
	self.ethClient = ethClient
	return self
}

func (self *AssetsCalculator) WithContractAbi(contractAbi map[string]*abi.ABI) *AssetsCalculator {
	self.contractAbi = contractAbi
	return self
}

func (self *AssetsCalculator) run() (err error) {
	for payload := range self.input {
		err = task.NewRetry().
			WithContext(self.Ctx).
			// Retries infinitely until success
			WithMaxElapsedTime(0).
			WithMaxInterval(self.Config.WarpySyncer.SyncerDepositBackoffInterval).
			WithAcceptableDuration(self.Config.WarpySyncer.SyncerDepositBackoffInterval * 2).
			WithOnError(func(err error, isDurationAcceptable bool) error {
				if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
					return backoff.Permanent(err)
				}

				self.monitor.GetReport().WarpySyncer.Errors.AssetsCalculatorFailures.Inc()
				self.Log.WithError(err).WithField("txId", payload.Transaction.Hash().String()).
					Warn("Could not calculate assets, retrying...")
				return err
			}).
			Run(func() error {
				var assets interface{}
				var assetsNames []string
				if slices.Contains(self.Config.WarpySyncer.StoreDepositWithdrawFunctions, payload.Method.Name) {
					assetsNames = self.Config.WarpySyncer.AssetsCalculatorWithdrawAssetsNames
					if payload.Transaction.To().String() == "0xA07c5b74C9B40447a954e1466938b865b6BBea36" {
						assets = nil
					} else {
						assets = self.getAssetsFromInput(assetsNames, payload.Input)
					}
				} else {
					assetsNames = self.Config.WarpySyncer.AssetsCalculatorDepositAssetsNames
					assets = self.getAssetsFromInput(assetsNames, payload.Input)
				}

				if assets == nil {
					assets, err = self.getAssetsFromLog(payload.Method.Name, payload.Transaction, assetsNames)
				}

				if err != nil {
					self.Log.WithError(err).Error("could not get assets from log")
					return nil
				}

				if assets == nil {
					err = errors.New("could not get assets from input and from log")
					self.Log.WithError(err)
					return nil
				}

				assetsVal := self.convertAssets(assets)

				if assetsVal == nil {
					err = errors.New("could not convert assets value")
					self.Log.WithError(err)
					return nil
				}

				var assetsInEth float64
				tokenName := eth.GetTokenName(payload.Transaction.To().String())
				if tokenName != "" {
					assetsInEth, err = self.convertTokenToEth(tokenName, assetsVal)
					if err != nil {
						return err
					}
				} else {
					assetsInEth = eth.WeiToEther(assetsVal)
				}

				select {
				case <-self.Ctx.Done():
					return nil
				case self.Output <- &SommelierTransactionPayload{
					Transaction: payload.Transaction,
					FromAddress: payload.FromAddress,
					Block:       payload.Block,
					Method:      payload.Method,
					ParsedInput: payload.ParsedInput,
					Input:       payload.Input,
					Assets:      assetsInEth,
				}:
				}

				return err
			})
	}
	return nil
}

func (self *AssetsCalculator) getAssetsFromInput(assetsNames []string, input map[string]interface{}) (assets interface{}) {
	for i, a := range assetsNames {
		if i == 0 {
			assets = input[a]
		} else {
			var assetsInterface map[string]interface{}
			assetsParsed, _ := json.Marshal(assets)
			err := json.Unmarshal(assetsParsed, &assetsInterface)
			if err != nil {
				self.Log.WithError(err).Error("Could not parse assets input")
				return nil
			}
			assets = assetsInterface[a]
		}
	}
	return
}

func (self *AssetsCalculator) convertAssets(assets interface{}) (assetsVal *big.Int) {
	switch assets := assets.(type) {
	case float64:
		assetsVal = new(big.Int)
		new(big.Float).SetFloat64(assets).Int(assetsVal)
	case *big.Int:
		assetsVal = assets
	default:
		self.Log.WithField("assets", assets).Error("Assets type unsupported")
		return nil
	}

	return
}

func (self *AssetsCalculator) getAssetsFromLog(methodName string, tx *types.Transaction, assetsNames []string) (assets interface{}, err error) {
	var logName string
	if slices.Contains(self.Config.WarpySyncer.StoreDepositWithdrawFunctions, methodName) {
		logName = self.Config.WarpySyncer.SyncerDepositWithdrawLog
	} else {
		logName = self.Config.WarpySyncer.SyncerDepositDepositLog
	}

	if logName != "" {
		receipt, err := self.ethClient.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			self.Log.WithError(err).WithField("tx_hash", tx.Hash().String()).Error("Could not get transaction receipt")
			return nil, err
		}
		log, err := eth.GetTransactionLog(receipt, self.contractAbi[tx.To().String()], logName)
		if err != nil {
			self.Log.WithError(err).WithField("tx_hash", tx.Hash().String()).Error("Could not parse log")
			return nil, err
		}
		assets = self.getAssetsFromInput(assetsNames, log)
	}

	return assets, nil
}

func (self *AssetsCalculator) convertTokenToEth(tokenName string, assetsVal *big.Int) (assetsInEth float64, err error) {
	tokenPriceInEth, err := eth.GetPriceInEth(tokenName)
	if err != nil {
		self.Log.WithError(err).Error("Could not get token price in ETH")
		return
	}

	decimals := self.Config.WarpySyncer.SyncerChain.Decimals()
	assetsValFloated, _ := big.NewFloat(0).SetInt(assetsVal).Float64()
	assetsInEth = (assetsValFloated / math.Pow(10, decimals)) * tokenPriceInEth
	return
}
