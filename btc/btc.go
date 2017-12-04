package btc

import "github.com/btcsuite/btcd/rpcclient"

func InitHandlers() (*rpcclient.Client, error) {
	return &rpcclient.Client{}, nil
}

type BtcTransaction struct {
	ID string
}
