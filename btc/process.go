package btc

import (
	"fmt"

	"github.com/Appscrunch/Multy-back/store"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"encoding/json"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"

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

var connCfg = &rpcclient.ConnConfig{
	Host:     "192.168.0.121:18334",
	User:     "multy",
	Pass:     "multy",
	Endpoint: "ws",
	Certificates: []byte(`-----BEGIN CERTIFICATE-----
MIICPDCCAZ2gAwIBAgIQf8XOycg2EQ8wHpXsZJSy7jAKBggqhkjOPQQDBDAjMREw
DwYDVQQKEwhnZW5jZXJ0czEOMAwGA1UEAxMFYW50b24wHhcNMTcxMTI2MTY1ODQ0
WhcNMjcxMTI1MTY1ODQ0WjAjMREwDwYDVQQKEwhnZW5jZXJ0czEOMAwGA1UEAxMF
YW50b24wgZswEAYHKoZIzj0CAQYFK4EEACMDgYYABAGuHzCFKsJwlFwmtx5QMT/r
YJ/ap9E2QlUsCnMUCn1ho0wLJkpIgNQWs1zcaKTMGZNpwwLemCHke9sX06h/MdAG
CwGf1CY5kafyl7dTTlmD10sBA7UD1RXDjYnmYQhB1Z1MUNXKWXe4jCv7DnWmFEnc
+s5N1NXJx1PNzx/EcsCkRJcMraNwMG4wDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB
/wQFMAMBAf8wSwYDVR0RBEQwQoIFYW50b26CCWxvY2FsaG9zdIcEfwAAAYcQAAAA
AAAAAAAAAAAAAAAAAYcEwKgAeYcQ/oAAAAAAAAByhcL//jB99jAKBggqhkjOPQQD
BAOBjAAwgYgCQgCfs9tYHA1nvU5HSdNeHSZCR1WziHYuZHmGE7eqAWQjypnVbFi4
pccvzDFvESf8DG4FVymK4E2T/RFnD9qUDiMzPQJCATkCMzSKcyYlsL7t1ZgQLwAK
UpQl3TYp8uTf+UWzBz0uoEbB4CFeE2G5ZzrVK4XWZK615sfVFSorxHOOZaLwZEEL
-----END CERTIFICATE-----`),
}

func RunProcess() error {
	fmt.Println("[DEBUG] RunProcess()")

	db, err := mgo.Dial("192.168.0.121:27017")
	fmt.Println(err)
	usersData = db.DB("cyberkek").C("users")

	mempoolRates = db.DB("BTCMempool").C("Rates")

	ntfnHandlers := rpcclient.NotificationHandlers{
		OnTxAcceptedVerbose: func(txDetails *btcjson.TxRawResult) {
			parseRawTransaction(txDetails)
		},
		OnRedeemingTx: func(transaction *btcutil.Tx, details *btcjson.BlockDetails) {
			log.Println("OnRedeemingTx ", transaction, details)
		},
		OnUnknownNotification: func(method string, params []json.RawMessage) {
			log.Println("OnUnknowNotification: ", method, params)
		},
		OnFilteredBlockDisconnected: func(height int32, header *wire.BlockHeader) {
			log.Printf("Block disconnected: %v (%d) %v",
				header.BlockHash(), height, header.Timestamp)
			//TODO update mem pool actual transactions

		},
		OnBlockConnected: func(hash *chainhash.Hash, height int32, t time.Time) {
			log.Printf("[DEBUG] OnBlockConnected: %v (%d) %v", hash, height, t)
			go getAndParseNewBlock(hash)
		},
	}

	rpcClient, err = rpcclient.New(connCfg, &ntfnHandlers)
	if err != nil {
		log.Printf("[ERR] RunProcess(): rpcclient.New %s\n", err.Error())
		return err
	}

	// Register for block connect and disconnect notifications.
	if err = rpcClient.NotifyBlocks(); err != nil {
		return err
	}
	log.Println("NotifyBlocks: Registration Complete")

	if err = rpcClient.NotifyNewTransactions(true); err != nil {
		return err
	}

	//When first launch here we are getting all mem pool transactions
	go getAllMempool()

	rpcClient.WaitForShutdown()
	return nil
}

var (
	//db           *mgo.Session
	mempoolRates *mgo.Collection
)

func getAllMempool() {
	rawMemPool, err := rpcClient.GetRawMempool()
	if err != nil {
		log.Println("ERR rpcClient.GetRawMempool [rawMemPool]: ", err.Error())
	}
	log.Printf("rawMemPoolSize: %d", len(rawMemPool))

	for _, txHash := range rawMemPool {
		go func(txHash *chainhash.Hash) {
			getRawTx(txHash)
		}(txHash)
	}
}

//Here we are getting transaction by hash
func getRawTx(hash *chainhash.Hash) {
	rawTx, err := rpcClient.GetRawTransactionVerbose(hash)
	if err != nil {
		log.Println("ERR GetRawTransactionVerbose: ", err.Error())
		//TODO
		return
	}
	go parseRawTransaction(rawTx)
}

func getAndParseNewBlock(hash *chainhash.Hash) {
	log.Printf("[DEBUG] getNewBlock()")
	blockMSG, err := rpcClient.GetBlock(hash)
	if err != nil {
		log.Println("[ERR] getAndParseNewBlock: ", err.Error())
	}

	// tx speed remover on block
	BlockTxHashes, err := blockMSG.TxHashes() // txHashes of all block tx's
	if err != nil {
		fmt.Printf("[ERR] getAndParseNewBlock(): TxHashes: %s\n", err.Error())
	}

	log.Println("[DEBUG] hash iteration logic transactions")

	var (
		user      store.User
		txHashStr string
	)
	for _, txHash := range BlockTxHashes {
		txHashStr = txHash.String()

		// !tx speed remover on block
		err := mempoolRates.Remove(bson.M{"hashtx": txHashStr})
		if err != nil {
			fmt.Println("[ERR] getNewBlock blockTxHases", err)
		}
		fmt.Println("[DEBUG] getNewBlock: removed:", txHash)

		blockTx, err := parseBlockTransaction(&txHash)
		if err != nil {
			log.Println("[ERR] parseBlockTransaction:  ", err.Error())
		}

		// !notify users that their transactions was applied in a block
		if err := usersData.Find(bson.M{"transactions.txhash": txHashStr}).One(&user); err != nil {
			log.Printf("[ERR] getAndParseNewBlock: usersData.Find: %s\n", err.Error())
			continue
		}

		if user.Wallets == nil {
			log.Println("[WARN] getAndParseNewBlock:  wallet is empty")
			continue
		}

		var output *store.BtcOutput
		var ok bool
		for _, wallet := range user.Wallets {
			for _, addr := range wallet.Adresses {
				if output, ok = blockTx.Outputs[addr.Address]; !ok {
					continue
				}
				// got output with our address; notify user about it
				log.Println("[DEBUG] getAndParseNewBlock: address=", addr)
				go addUserTransactionsToDB(user.UserID, output)
				chToClient <- CreateBtcTransactionWithUserID(addr.Address, user.UserID, txOut+" block", txHashStr, output.Amount)
			}
		}
	}
	log.Println("[DEBUG] hash iteration logic transactions done")

	for _, tx := range blockMSG.Transactions {
		for index, memTx := range memPool {
			if memTx.hash == tx.TxHash().String() {
				//TODO remove transaction from mempool
				//TODO update fee rates table
				//TODO check if tx is of our client
				//TODO is so -> notify client
				memPool = append(memPool[:index], memPool[index+1:]...)
			}
		}
	}

	log.Println("[DEBUG] getNewBlock() done")
}

func serealizeBTCTransaction(currentTx *btcjson.TxRawResult) *store.BTCTransaction {
	blockTx := store.BTCTransaction{
		Hash: currentTx.Hash,
		Txid: currentTx.Txid,
		Time: time.Now(),
	}

	log.Printf("[DEBUG] blocktx=%+v\n", blockTx)
	outputsAll := currentTx.Vout

	outputsMultyUsers := make(map[string]*store.BtcOutput, 0)
	for _, output := range outputsAll {
		addressesOuts := output.ScriptPubKey.Addresses
		if len(addressesOuts) == 0 {
			log.Println("[WARN] serealizeTransaction: len(addressesOuts)==0")
			continue
		}
		outputsMultyUsers[addressesOuts[0]] = &store.BtcOutput{
			Address:     addressesOuts[0],
			Amount:      output.Value,
			TxIndex:     output.N,
			TxOutScript: output.ScriptPubKey.Hex,
		}
	}
	blockTx.Outputs = outputsMultyUsers
	return &blockTx
}

func addUserTransactionsToDB(userID string, output *store.BtcOutput) {
	log.Print("[DEBUG] addUserTransactionsToDB")

	sel := bson.M{"userID": userID}
	update := bson.M{"$push": bson.M{"transactions": output}}
	err := usersData.Update(sel, update)
	if err != nil {
		fmt.Printf("[ERR] push transaction to db: %s\n", err.Error())
	}
	log.Print("[DEBUG] addUserTransactionsToDB done")
}

func parseBlockTransaction(txHash *chainhash.Hash) (*store.BTCTransaction, error) {
	log.Printf("[DEBUG] parseBlockTransaction()")
	currentRaw, err := rpcClient.GetRawTransactionVerbose(txHash)
	if err != nil {
		fmt.Printf("[ERR] parseBlockTransaction: %s\n", err.Error())
		return nil, err
	}

	blockTx := serealizeBTCTransaction(currentRaw)

	log.Printf("[DEBUG] done parseBlockTransaction: %+v\n", blockTx.Outputs)
	// log.Printf("\nMulty-New Tx Found hash: %s, id: %s \n amount: %f , fee: %f , feeRate: %d \n Inputs: %v \n OutPuts: %v \Multy-the best wallet*******************", memPoolTx.hash, memPoolTx.txid, memPoolTx.amount, memPoolTx.fee, memPoolTx.feeRate, memPoolTx.inputs, memPoolTx.outputs)

	return blockTx, nil
}
