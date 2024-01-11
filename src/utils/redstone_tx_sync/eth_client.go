package redstone_tx_sync

import (
	"errors"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

type EthUrl int64

const (
	Avax EthUrl = iota
)

func (ethUrl EthUrl) String() (url string, err error) {
	switch ethUrl {
	case Avax:
		url = "https://api.avax.network/ext/bc/C/rpc"
		return
	}

	err = errors.New("ETH url unknown")
	return
}

func GetEthClient(log *logrus.Entry, url EthUrl) (client *ethclient.Client, err error) {
	ethUrl, err := url.String()
	if err != nil {
		log.WithError(err).Error("ETH url unknown")
		return
	}

	client, err = ethclient.Dial(ethUrl)
	if err != nil {
		log.WithError(err).Error("Cannot get ETH client")
		return
	}

	return
}
