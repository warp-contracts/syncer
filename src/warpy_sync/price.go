package warpy_sync

import (
	"fmt"
)

type Prices struct {
	Bnb float64
	Btc float64
	Sei float64
}

func (p *Prices) GetByTokenName(name string) (price float64, err error) {
	switch name {
	case "binancecoin":
		price = p.Bnb
		return
	case "bitcoin":
		price = p.Btc
		return
	case "sei":
		price = p.Sei
		return
	}
	return 0, fmt.Errorf("cannot get price. token name %s not recognized", name)
}

func (p *Prices) SetByTokenName(name string, price float64) error {
	switch name {
	case "binancecoin":
		p.Bnb = price
		return nil
	case "bitcoin":
		p.Btc = price
		return nil
	case "sei":
		p.Sei = price
		return nil
	}
	return fmt.Errorf("cannot set price. token name %s not recognized", name)
}
