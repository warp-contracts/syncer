package initsequencer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"

	"github.com/warp-contracts/sequencer/x/sequencer/types"
)

const (
	NETWORK_FOLDER          = "network"
	GENESIS_FOLDER          = "genesis"
	PREV_SORT_KEYS_FILE     = "prev_sort_keys.json"
	LAST_ARWEAVE_BLOCK_FILE = "arweave_block.json"
)

type Writer struct {
	*task.Task

	sequencerRepoPath string
	env               string
	db                *gorm.DB
	input             chan *listener.Payload
	Output            chan struct{}
	lastSyncedBlock   model.State
}

func NewWriter(config *config.Config) (self *Writer) {
	self = new(Writer)
	self.Output = make(chan struct{}, 1)

	self.Task = task.NewTask(config, "writer").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Writer) WithSequencerRepoPath(sequencerRepoPath string) *Writer {
	self.sequencerRepoPath = sequencerRepoPath
	return self
}

func (self *Writer) WithEnv(env string) *Writer {
	self.env = env
	return self
}

func (self *Writer) WithDB(db *gorm.DB) *Writer {
	self.db = db
	return self
}

func (self *Writer) WithInput(input chan *listener.Payload) *Writer {
	self.input = input
	return self
}

func (self *Writer) WithLastSyncedBlock(lastSyncedBlock model.State) *Writer {
	self.lastSyncedBlock = lastSyncedBlock
	return self
}

func (self *Writer) run() (err error) {
	err = self.fetchPrevSortKeys()
	if err != nil {
		return
	}

	err = self.fetchLastArweaveBlock()
	if err != nil {
		return
	}

	return
}

func (self *Writer) fetchPrevSortKeys() (err error) {
	var prevSortKeys []*types.PrevSortKey
	err = self.db.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			return self.db.Table(model.TableInteraction).
				Select("contract_id as contract, max(sort_key) as sort_key").
				Where("contract_id != ''").
				Group("contract_id").
				Scan(&prevSortKeys).
				Error
		})
	if err != nil {
		return err
	}

	keysJson, err := json.Marshal(prevSortKeys)
	if err != nil {
		return
	}

	err = self.writeToConfigFile(PREV_SORT_KEYS_FILE, keysJson)
	if err != nil {
		return
	}

	self.Log.WithField("number of keys", len(prevSortKeys)).Debug("Prev sort keys saved to file")
	return
}

func (self *Writer) fetchLastArweaveBlock() (err error) {
	for payload := range self.input {

		fmt.Println("GET PAYLOAD")

		lastBlockInfo := &types.ArweaveBlockInfo{
			Height:    uint64(self.lastSyncedBlock.FinishedBlockHeight),
			Timestamp: uint64(self.lastSyncedBlock.FinishedBlockTimestamp),
			Hash:      self.lastSyncedBlock.FinishedBlockHash.Base64(),
		}

		nextBlockInfo := &types.ArweaveBlockInfo{
			Height:    uint64(payload.BlockHeight),
			Timestamp: uint64(payload.BlockTimestamp),
			Hash:      payload.BlockHash.Base64(),
		}

		nextBlock := &types.NextArweaveBlock{
			BlockInfo:    nextBlockInfo,
			Transactions: transactions(payload),
		}

		block := types.GenesisArweaveBlock{
			LastArweaveBlock: lastBlockInfo,
			NextArweaveBlock: nextBlock,
		}

		blockJson, err := json.Marshal(block)
		if err != nil {
			return err
		}

		err = self.writeToConfigFile(LAST_ARWEAVE_BLOCK_FILE, blockJson)
		if err != nil {
			return err
		}

		self.Log.
			WithField("last synced block height", lastBlockInfo.Height).
			WithField("next block height", nextBlockInfo.Height).
			Debug("Arweave block saved to file")
		self.Output <- struct{}{}
	}

	return
}

func transactions(payload *listener.Payload) []*types.ArweaveTransaction {
	txs := make([]*types.ArweaveTransaction, 0, len(payload.Transactions))
	for _, tx := range payload.Transactions {
		contract, found := tx.GetTag(smartweave.TagContractTxId)
		if !found {
			continue
		}

		txs = append(txs, &types.ArweaveTransaction{
			Id:       tx.ID.Base64(),
			Contract: contract,
			SortKey:  warp.CreateSortKey(tx.ID, payload.BlockHeight, payload.BlockHash),
		})
	}

	// sort transactions by sort key
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].SortKey < txs[j].SortKey
	})
	return txs
}

func (self *Writer) writeToConfigFile(filePath string, jsonData []byte) error {
	fileFullPath := filepath.Join(self.sequencerRepoPath, NETWORK_FOLDER, self.env, GENESIS_FOLDER, filePath)
	err := os.WriteFile(fileFullPath, jsonData, 0644)
	if err != nil {
		return err
	}
	return nil
}
