package initsequencer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"

	"github.com/warp-contracts/sequencer/x/sequencer/types"
)

const (
	CONFIG_PATH_FORMAT      = "network/local/sequencer-%d/config/"
	LAST_SORT_KEYS_FILE     = "last_sort_keys.json"
	LAST_ARWEAVE_BLOCK_FILE = "last_arweave_block.json"
	NUMBER_OF_NODES         = 3
)

type Writer struct {
	*task.Task

	sequencerRepoPath string
	db                *gorm.DB
	input             chan *arweave.Block
	Output            chan interface{}
}

func NewWriter(config *config.Config, sequencerRepoPath string) (self *Writer) {
	self = new(Writer)
	self.Output = make(chan interface{}, 1)

	self.Task = task.NewTask(config, "writer").
		WithSubtaskFunc(self.run)

	return
}

func (self *Writer) WithSequencerRepoPath(sequencerRepoPath string) *Writer {
	self.sequencerRepoPath = sequencerRepoPath
	return self
}

func (self *Writer) WithDB(db *gorm.DB) *Writer {
	self.db = db
	return self
}

func (self *Writer) WithInput(input chan *arweave.Block) *Writer {
	self.input = input
	return self
}

func (self *Writer) run() (err error) {
	err = self.fetchLastSortKeys()
	if err != nil {
		return
	}

	err = self.fetchLastArweaveBlock()
	if err != nil {
		return
	}

	return
}

func (self *Writer) fetchLastSortKeys() (err error) {
	var lastSortKeys []*types.LastSortKey
	err = self.db.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			return self.db.Table(model.TableInteraction).
				Select("contract_id as contract, max(sort_key) as sort_key").
				Where("contract_id != ''").
				Group("contract_id").
				Scan(&lastSortKeys).
				Error
		})
	if err != nil {
		return err
	}

	keysJson, err := json.Marshal(lastSortKeys)
	if err != nil {
		return
	}

	err = self.writeToConfigFile(LAST_SORT_KEYS_FILE, keysJson)
	if err != nil {
		return
	}

	self.Log.WithField("number of keys", len(lastSortKeys)).Debug("Last sort keys saved to files")
	return
}

func (self *Writer) fetchLastArweaveBlock() (err error) {
	for block := range self.input {
		blockInfo := &types.ArweaveBlockInfo{
			Height:    uint64(block.Height),
			Timestamp: uint64(block.Timestamp),
			Hash:      block.IndepHash.Base64(),
		}

		blockJson, err := json.Marshal(blockInfo)
		if err != nil {
			return err
		}

		err = self.writeToConfigFile(LAST_ARWEAVE_BLOCK_FILE, blockJson)
		if err != nil {
			return err
		}

		self.Log.WithField("height", blockInfo.Height).Debug("Last Arweave block saved to files")
		self.Output <- struct{}{}
	}

	return
}

func (self *Writer) writeToConfigFile(filePath string, jsonData []byte) error {
	for i := 0; i < NUMBER_OF_NODES; i++ {
		filePath := filepath.Join(self.sequencerRepoPath, fmt.Sprintf(CONFIG_PATH_FORMAT, i), filePath)
		err := os.WriteFile(filePath, jsonData, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
