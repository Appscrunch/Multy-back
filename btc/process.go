package btc

import (
	"github.com/btcsuite/btcd/rpcclient"
	mgo "gopkg.in/mgo.v2"

	"log"
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

var usersData *mgo.Collection

func (btcC *BTCClient) RunProcess() error {

	/*rpcClient, err = btcC.rpcclient.New(btcC.connCfg, &ntfnHandlers)
	if err != nil {
		log.Printf("[ERR] RunProcess(): rpcclient.New %s\n", err.Error())
		return err
	}*/

	// Register for block connect and disconnect notifications.
	if err := btcC.rpcClient.NotifyBlocks(); err != nil {
		return err
	}
	log.Println("NotifyBlocks: Registration Complete")

	// Register for new transaction in mempool notifications.
	if err := btcC.rpcClient.NotifyNewTransactions(true); err != nil {
		return err
	}
	log.Println("NotifyNewTransactions: Registration Complete")

	btcC.rpcClient.WaitForShutdown()
	return nil
}
