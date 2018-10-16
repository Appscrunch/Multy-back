/*
Copyright 2019 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/
package eth

import (
	"fmt"
	"github.com/Multy-io/Multy-back/retry"
	"sync"

	"google.golang.org/grpc"
	mgo "gopkg.in/mgo.v2"

	"github.com/Multy-io/Multy-back/currencies"
	pb "github.com/Multy-io/Multy-back/node-streamer/eth"
	"github.com/Multy-io/Multy-back/store"
	nsq "github.com/bitly/go-nsq"
	"github.com/jekabolt/slf"
)

// ETHConn is a main struct of package
type ETHConn struct {
	NsqProducer      *nsq.Producer // a producer for sending data to clients
	CliTest          pb.NodeCommuunicationsClient
	CliMain          pb.NodeCommuunicationsClient
	WatchAddressTest chan pb.WatchAddress
	WatchAddressMain chan pb.WatchAddress
	// Mempool          *map[string]int
	// MempoolTest      *map[string]int

	Mempool     sync.Map
	MempoolTest sync.Map

	VersionMain store.NodeVersion
	VersionTest store.NodeVersion

	// M     *sync.Mutex
	// MTest *sync.Mutex
}

var log = slf.WithContext("eth")

//InitHandlers init nsq mongo and ws connection to node
// return main client , test client , err
func InitHandlers(dbConf *store.Conf, coinTypes []store.CoinType, nsqAddr string, retrier retry.Retrier) (*ETHConn, error) {
	//declare pacakge struct
	cli := &ETHConn{
		Mempool:     sync.Map{},
		MempoolTest: sync.Map{},
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
		log.Errorf("RunProcess: can't connect to DB: %s", err.Error())
		return cli, fmt.Errorf("mgo.Dial: %s", err.Error())
	}
	log.Infof("InitHandlers: mgo.Dial: √")

	usersData = db.DB(dbConf.DBUsers).C(store.TableUsers) // all db tables
	exRate = db.DB(dbConf.DBStockExchangeRate).C("TableStockExchangeRate")

	// main
	txsData = db.DB(dbConf.DBTx).C(dbConf.TableTxsDataETHMain)

	// test
	txsDataTest = db.DB(dbConf.DBTx).C(dbConf.TableTxsDataETHTest)

	//restore state
	restoreState = db.DB(dbConf.DBRestoreState).C(dbConf.TableState)

	// setup main net
	urlMain, err := fethCoinType(coinTypes, currencies.Ether, currencies.ETHMain)
	if err != nil {
		return cli, fmt.Errorf("fethCoinType: %s", err.Error())
	}

	var cliMain pb.NodeCommuunicationsClient
	err = retrier.Do(log.WithField("net", "ETH MAIN"), func () (err error) {
		cliMain, err = initGrpcClient(urlMain)
		return
	})
	if err != nil {
		return cli, fmt.Errorf("initGrpcClient: %s", err.Error())
	}
	setGRPCHandlers(cliMain, cli.NsqProducer, currencies.ETHMain, cli.WatchAddressMain, cli.Mempool)

	cli.CliMain = cliMain
	log.Infof("InitHandlers: initGrpcClient: Main: √")

	// setup testnet
	urlTest, err := fethCoinType(coinTypes, currencies.Ether, currencies.ETHTest)
	if err != nil {
		return cli, fmt.Errorf("fethCoinType: %s", err.Error())
	}
	var cliTest pb.NodeCommuunicationsClient
	err = retrier.Do(log.WithField("net", "ETH TEST"), func () (err error) {
		cliMain, err = initGrpcClient(urlTest)
		return
	})

	if err != nil {
		return cli, fmt.Errorf("initGrpcClient: %s", err.Error())
	}
	setGRPCHandlers(cliTest, cli.NsqProducer, currencies.ETHTest, cli.WatchAddressTest, cli.MempoolTest)

	cli.CliTest = cliTest
	log.Infof("InitHandlers: initGrpcClient: Test: √")

	return cli, nil
}

func initGrpcClient(url string) (pb.NodeCommuunicationsClient, error) {
	conn, err := grpc.Dial(url, grpc.WithInsecure())
	if err != nil {
		log.WithCaller(slf.CallerShort).WithError(err).Error("Failed to establish grpc connection")
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

// BtcTransaction stuct for ws notifications
type Transaction struct {
	TransactionType int    `json:"transactionType"`
	Amount          string `json:"amount"`
	TxID            string `json:"txid"`
	Address         string `json:"address"`
}

// BtcTransactionWithUserID sub-stuct for ws notifications
type TransactionWithUserID struct {
	NotificationMsg *Transaction
	UserID          string
}
