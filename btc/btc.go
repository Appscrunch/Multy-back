package btc

import (
	"fmt"
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

const (
	txIn  = "incoming"
	txOut = "outcoming"
)

type BTCClient struct {
	chToClient chan BtcTransactionWithUserID
	rpcClient  *rpcclient.Client
	rpcConf    *rpcclient.ConnConfig
}

func (btcC *BTCClient) simulateSendNewTransactions() {
	for {
		time.Sleep(time.Second * 2)
		b := BtcTransactionWithUserID{
			NotificationMsg: &BtcTransaction{
				Amount: 5,
			},
			UserID: "555",
		}

		btcC.chToClient <- b
	}
}

func InitHandlers(rpcConf *rpcclient.ConnConfig) (*rpcclient.Client, chan BtcTransactionWithUserID, error) {
	// go simulateSendNewTransactions()
	fmt.Println("[DEBUG] RunProcess()")

	//var err error
	db, err := mgo.Dial("192.168.0.121:27017")
	fmt.Println(err)

	usersData = db.DB("userDB").C("userCollection")

	clientBTC := BTCClient{
		rpcConf: rpcConf,
	}

	ntfnHandlers := rpcclient.NotificationHandlers{
		OnBlockConnected: func(hash *chainhash.Hash, height int32, t time.Time) {
			log.Printf("[DEBUG] OnBlockConnected: %v (%d) %v", hash, height, t)
			go clientBTC.getAndParseNewBlock(hash)
		},
		OnTxAcceptedVerbose: func(txDetails *btcjson.TxRawResult) {
			log.Printf("[DEBUG] OnTxAcceptedVerbose: new transaction id = %v", txDetails.Txid)
			// notify on new in
			// notify on new out
			parseMempoolTransaction(txDetails)
		},
	}

	rpcClient, err := rpcclient.New(rpcConf, &ntfnHandlers)
	if err != nil {
		log.Printf("[ERR] InitHandlers: %s\n", err.Error())
		return nil, nil, err
	}

	chToClient := make(chan BtcTransactionWithUserID, 0)

	clientBTC.chToClient = chToClient
	clientBTC.rpcClient = rpcClient

	go clientBTC.RunProcess()
	return rpcClient, chToClient, nil
}

type BtcTransaction struct {
	TransactionType string  `json:"transactionType"`
	Amount          float64 `json:"amount"`
	TxID            string  `json:"txid"`
	Address         string  `json:"address"`
}

type BtcTransactionWithUserID struct {
	NotificationMsg *BtcTransaction
	UserID          string
}
