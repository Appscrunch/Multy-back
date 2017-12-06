package btc

import (
	"log"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
)

// Dirty hack - this will be wrapped to a struct
var (
	rpcClient  = &rpcclient.Client{}
	chToClient chan BtcTransactionWithUserID
)

func simulateSendNewTransactions() {
	for {
		time.Sleep(time.Second * 2)
		b := BtcTransactionWithUserID{
			NotificationMsg: &BtcTransaction{
				Amount: 5,
			},
			UserID: "555",
		}
		log.Printf("sending  new transaction: %+v\n", b)
		chToClient <- b
	}
}

func InitHandlers() (*rpcclient.Client, chan BtcTransactionWithUserID, error) {
	chToClient = make(chan BtcTransactionWithUserID, 0)
	go simulateSendNewTransactions()

	return rpcClient, chToClient, nil
}

type BtcTransaction struct {
	TransactionType string `json:"transactionType"`
	Amount          int64  `json:"amount"`
	TxID            int64  `json:"txid"`
}

type BtcTransactionWithUserID struct {
	NotificationMsg *BtcTransaction
	UserID          string
}
