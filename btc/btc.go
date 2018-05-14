package btc

/*
Copyright 2019 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/

import (
	"fmt"
	"sync"

	"google.golang.org/grpc"
	mgo "gopkg.in/mgo.v2"

	"github.com/Appscrunch/Multy-back/currencies"
	pb "github.com/Appscrunch/Multy-back/node-streamer/btc"
	"github.com/Appscrunch/Multy-back/store"
	nsq "github.com/bitly/go-nsq"
	"github.com/jekabolt/slf"
)

// Conn is a main struct of package
type Conn struct {
	NsqProducer      *nsq.Producer // A producer for sending data to clients
	CliTest          pb.NodeCommuunicationsClient
	CliMain          pb.NodeCommuunicationsClient
	WatchAddressTest chan pb.WatchAddress
	WatchAddressMain chan pb.WatchAddress
	BTCMempool       *map[string]int
	BTCMempoolTest   *map[string]int
	Resync           *map[string]bool
	ResyncM          *sync.Mutex
	M                *sync.Mutex
	MTest            *sync.Mutex
}

var log = slf.WithContext("btc")

// InitHandlers init nsq mongo and ws connection to node
// return main client , test client , err
func InitHandlers(dbConf *store.Conf, coinTypes []store.CoinType, nsqAddr string) (*Conn, error) {
	// Declare pacakge struct
	cli := &Conn{
		BTCMempool:     &map[string]int{},
		BTCMempoolTest: &map[string]int{},
		Resync:         &map[string]bool{},
		ResyncM:        &sync.Mutex{},
		M:              &sync.Mutex{},
		MTest:          &sync.Mutex{},
	}

	cli.WatchAddressMain = make(chan pb.WatchAddress)
	cli.WatchAddressTest = make(chan pb.WatchAddress)

	config := nsq.NewConfig()
	p, err := nsq.NewProducer(nsqAddr, config)
	if err != nil {
		return cli, fmt.Errorf("nsq producer: %s", err.Error())
	}

	cli.NsqProducer = p
	log.Infof("InitHandlers: nsq.NewProducer: √")

	addr := []string{dbConf.Address}

	mongoDBDial := &mgo.DialInfo{
		Addrs:    addr,
		Username: dbConf.Username,
		Password: dbConf.Password,
	}

	db, err := mgo.DialWithInfo(mongoDBDial)
	if err != nil {
		return nil, err
	}
	if err != nil {
		log.Errorf("RunProcess: can't connect to DB: %s", err.Error())
		return cli, fmt.Errorf("mgo.Dial: %s", err.Error())
	}
	log.Infof("InitHandlers: mgo.Dial: √")

	usersData = db.DB(dbConf.DBUsers).C(store.TableUsers) // All db tables
	exRate = db.DB(dbConf.DBStockExchangeRate).C("TableStockExchangeRate")

	// Main
	txsData = db.DB(dbConf.DBTx).C(dbConf.TableTxsDataBTCMain)
	spendableOutputs = db.DB(dbConf.DBTx).C(dbConf.TableSpendableOutputsBTCMain)
	spentOutputs = db.DB(dbConf.DBTx).C(dbConf.TableSpentOutputsBTCMain)

	// Test
	txsDataTest = db.DB(dbConf.DBTx).C(dbConf.TableTxsDataBTCTest)
	spendableOutputsTest = db.DB(dbConf.DBTx).C(dbConf.TableSpendableOutputsBTCTest)
	spentOutputsTest = db.DB(dbConf.DBTx).C(dbConf.TableSpentOutputsBTCTest)

	// Setup main net
	urlMain, err := fethCoinType(coinTypes, currencies.Bitcoin, currencies.Main)
	if err != nil {
		return cli, fmt.Errorf("fethCoinType: %s", err.Error())
	}

	cliMain, err := initGrpcClient(urlMain)
	if err != nil {
		return cli, fmt.Errorf("initGrpcClient: %s", err.Error())
	}

	setGRPCHandlers(cliMain, cli.NsqProducer, currencies.Main, cli.WatchAddressMain, cli.BTCMempool, cli.M, cli.Resync, cli.ResyncM)

	cli.CliMain = cliMain
	log.Infof("InitHandlers: initGrpcClient: Main: √")

	// Setup testnet
	urlTest, err := fethCoinType(coinTypes, currencies.Bitcoin, currencies.Test)
	if err != nil {
		return cli, fmt.Errorf("fethCoinType: %s", err.Error())
	}
	cliTest, err := initGrpcClient(urlTest)
	if err != nil {
		return cli, fmt.Errorf("initGrpcClient: %s", err.Error())
	}
	setGRPCHandlers(cliTest, cli.NsqProducer, currencies.Test, cli.WatchAddressTest, cli.BTCMempoolTest, cli.MTest, cli.Resync, cli.ResyncM)

	cli.CliTest = cliTest
	log.Infof("InitHandlers: initGrpcClient: Test: √")

	return cli, nil
}

func initGrpcClient(url string) (pb.NodeCommuunicationsClient, error) {
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		log.Errorf("initGrpcClient: grpc.Dial: %s", err.Error())
		return nil, err
	}

	// Create a new  client
	client := pb.NewNodeCommuunicationsClient(conn)
	return client, nil
}

func fethCoinType(coinTypes []store.CoinType, currencyID, networkID int) (string, error) {
	for _, ct := range coinTypes {
		if ct.СurrencyID == currencyID && ct.NetworkID == networkID {
			return ct.GRPCUrl, nil
		}
	}
	return "", fmt.Errorf("fethCoinType: no such coin in config")
}

// // BtcTransaction stuct for ws notifications
// type BtcTransaction struct {
// 	TransactionType int    `json:"transactionType"`
// 	Amount          int64  `json:"amount"`
// 	TxID            string `json:"txid"`
// 	Address         string `json:"address"`
// }

// // BtcTransactionWithUserID sub-stuct for ws notifications
// type BtcTransactionWithUserID struct {
// 	NotificationMsg *BtcTransaction
// 	UserID          string
// }
