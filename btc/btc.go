package btc

import (
	"fmt"

	mgo "gopkg.in/mgo.v2"

	"github.com/Appscrunch/Multy-back/store"
	"github.com/KristinaEtc/slf"
	nsq "github.com/bitly/go-nsq"
	"github.com/btcsuite/btcd/rpcclient"
)

const (
	txInMempool  = "incoming from mempool"
	txOutMempool = "outcoming from mempool"
	txInBlock    = "incoming from block"
	txOutBlock   = "outcoming from block"

	// TopicTransaction is a topic for sending notifies to clients
	TopicTransaction = "btcTransactionUpdate"
)

// Dirty hack - this will be wrapped to a struct
var (
	rpcClient = &rpcclient.Client{}

	nsqProducer *nsq.Producer // a producer for sending data to clients
	rpcConf     *rpcclient.ConnConfig
)

var log = slf.WithContext("btc")

func InitHandlers(certFromConf string, dbConf *store.Conf) (*rpcclient.Client, error) {
	config := nsq.NewConfig()
	p, err := nsq.NewProducer("127.0.0.1:4150", config)
	if err != nil {
		return nil, fmt.Errorf("nsq producer: %s", err.Error())
	}
	nsqProducer = p

	Cert = certFromConf
	connCfg.Certificates = []byte(Cert)
	log.Infof("cert=%s\n", Cert)

	db, err := mgo.Dial("localhost:27017")
	if err != nil {
		log.Errorf("RunProcess: Cand connect to DB: %s", err.Error())
		return nil, err
	}

	usersData = db.DB(dbConf.DBUsers).C(store.TableUsers) // all db tables
	mempoolRates = db.DB(dbConf.DBFeeRates).C(store.TableFeeRates)
	txsData = db.DB(dbConf.DBTx).C(store.TableBTC)
	exRate = db.DB("DBStockExchangeRate-text").C("TableStockExchangeRate")

	go RunProcess()
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
