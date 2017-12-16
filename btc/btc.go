package btc

import (
	"fmt"
	"log"
	"time"

	"github.com/bitly/go-nsq"

	"github.com/Appscrunch/Multy-back/store"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

const (
	txIn  = "incoming"
	txOut = "outcoming"

	topicTransaction = "btcTransactionUpdate"
)

type BTCClient struct {
	nsqProducer *nsq.Producer
	rpcClient   *rpcclient.Client
	rpcConf     *rpcclient.ConnConfig
}

func InitHandlers(dbStore store.UserStore, rpcConf *rpcclient.ConnConfig) (*rpcclient.Client, error) {
	fmt.Println("[DEBUG] RunProcess()")

	usersData = dbStore.UserDataCollection()

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
		return nil, err
	}

	clientBTC.rpcClient = rpcClient

	config := nsq.NewConfig()
	p, err := nsq.NewProducer("127.0.0.1:4150", config)
	if err != nil {
		return nil, fmt.Errorf("nsq producer: %s", err.Error())
	}
	clientBTC.nsqProducer = p

	go clientBTC.RunProcess()
	return rpcClient, nil
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
