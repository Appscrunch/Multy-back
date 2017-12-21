package btc

import (
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/rpcclient"
	mgo "gopkg.in/mgo.v2"

	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

type MultyMempoolTx struct {
	hash    string
	inputs  []MultyAddress
	outputs []MultyAddress
	amount  float64
	fee     float64
	size    int32
	feeRate int32
	txid    string
}

type MultyAddress struct {
	address []string
	amount  float64
}

var memPool []MultyMempoolTx

type rpcClientWrapper struct {
	*rpcclient.Client
}

var (
	usersData    *mgo.Collection
	mempoolRates *mgo.Collection
	txsData      *mgo.Collection
)

var Cert = `-----BEGIN CERTIFICATE-----
MIIChjCCAeigAwIBAgIRAIk9dSekS8kr907yIOLdYHkwCgYIKoZIzj0EAwQwOzER
MA8GA1UEChMIZ2VuY2VydHMxJjAkBgNVBAMTHVVidW50dS0xNzEwLWFydGZ1bC02
NC1taW5pbWFsMB4XDTE3MTIwNzEwMTAyNloXDTI3MTIwNjEwMTAyNlowOzERMA8G
A1UEChMIZ2VuY2VydHMxJjAkBgNVBAMTHVVidW50dS0xNzEwLWFydGZ1bC02NC1t
aW5pbWFsMIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQAxxPJzJr2GO/KHwKwSvpS
365ScP47ChcunVxRqikDv55tiYgsJBj+OAd9CJPrTlTPRW1OPkAf6hg9Pv4LPYBg
aBoAhUQkE7YiPjbJkZhg98DL0RaTlebcXWHE0UQ6SNXE8Or3MZNZljL/HTPrPvsl
ezVgytE1aJsK3fmZpFp6Tolj3/GjgYkwgYYwDgYDVR0PAQH/BAQDAgKkMA8GA1Ud
EwEB/wQFMAMBAf8wYwYDVR0RBFwwWoIdVWJ1bnR1LTE3MTAtYXJ0ZnVsLTY0LW1p
bmltYWyCCWxvY2FsaG9zdIcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAAYcEWMYvcIcQ
/oAAAAAAAAAW2un//u9nhTAKBggqhkjOPQQDBAOBiwAwgYcCQgDoEBs8oe8QvOZP
RZo+Hck5JZZhBtHOWZRsgi/GsWOuLvLJiJxnxWDUQQkuJewlWugMDpH0jTqf9Sm/
Tc9SOTFf7QJBbTyfcozACphA7sn1LrIH7Savrw5CnLKgCmfDdCgmnyM3GbK+soId
wcNvBlvz4dHHRbDGr5U019eArX1HF6JvY4k=
-----END CERTIFICATE-----`

var connCfg = &rpcclient.ConnConfig{
	Host:         "localhost:18334",
	User:         "multy",
	Pass:         "multy",
	Endpoint:     "ws",
	Certificates: []byte(Cert),

	HTTPPostMode: false, // Bitcoin core only supports HTTP POST mode
	DisableTLS:   false, // Bitcoin core does not provide TLS by default

}

func RunProcess() error {
	log.Info("Run Process")

	db, err := mgo.Dial("localhost:27017")

	if err != nil {
		log.Errorf("RunProcess: Cand connect to DB: %s", err.Error())
		return err
	}

	usersData = db.DB("userDB").C("userCollection") // all db tables
	mempoolRates = db.DB("BTCMempool").C("Rates")
	txsData = db.DB("Tx").C("BTC")

	// Drop collection on every new start of application
	err = mempoolRates.DropCollection()
	if err != nil {
		log.Errorf("RunProcess:mempoolRates.DropCollection:%s", err.Error())
	}

	ntfnHandlers := rpcclient.NotificationHandlers{
		OnBlockConnected: func(hash *chainhash.Hash, height int32, t time.Time) {
			log.Debugf("OnBlockConnected: %v (%d) %v", hash, height, t)
			go notifyNewBlockTx(hash)
			go blockTransactions(hash)
			go blockConfirmations(hash)
		},
		OnTxAcceptedVerbose: func(txDetails *btcjson.TxRawResult) {
			log.Debugf("OnTxAcceptedVerbose: new transaction id = %v", txDetails.Txid)
			// notify on new in
			// notify on new out
			go parseMempoolTransaction(txDetails)
			//add every new tx from mempool to db
			//feeRate
			go newTxToDB(txDetails)

			go mempoolTransaction(txDetails)

		},
	}

	rpcClient, err = rpcclient.New(connCfg, &ntfnHandlers)
	if err != nil {
		log.Errorf("RunProcess(): rpcclient.New %s\n", err.Error())
		return err
	}

	// Register for block connect and disconnect notifications.
	if err = rpcClient.NotifyBlocks(); err != nil {
		return err
	}
	log.Info("NotifyBlocks: Registration Complete")

	// Register for new transaction in mempool notifications.
	if err = rpcClient.NotifyNewTransactions(true); err != nil {
		return err
	}
	log.Info("NotifyNewTransactions: Registration Complete")

	// get all mempool and append to db
	go getAllMempool()

	rpcClient.WaitForShutdown()
	return nil
}
