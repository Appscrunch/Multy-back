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

//Here we parsing transaction by getting inputs and outputs addresses
func parseRawTransaction(inTx *btcjson.TxRawResult) error {
	memPoolTx := MultyMempoolTx{size: inTx.Size, hash: inTx.Hash, txid: inTx.Txid}

	inputs := inTx.Vin

	var inputSum float64 = 0
	var outputSum float64 = 0

	for j := 0; j < len(inputs); j++ {
		input := inputs[j]

		inputNum := input.Vout

		txCHash, errCHash := chainhash.NewHashFromStr(input.Txid)

		if errCHash != nil {
			log.Fatal(errCHash)
		}

		oldTx, err := rpcClient.GetRawTransactionVerbose(txCHash)
		if err != nil {
			log.Println("ERR GetRawTransactionVerbose [old]: ", err.Error())
			return err
		}

		oldOutputs := oldTx.Vout

		inputSum += oldOutputs[inputNum].Value

		addressesInputs := oldOutputs[inputNum].ScriptPubKey.Addresses

		inputAdr := MultyAddress{addressesInputs, oldOutputs[inputNum].Value}

		memPoolTx.inputs = append(memPoolTx.inputs, inputAdr)
	}

	outputs := inTx.Vout

	var txOutputs []MultyAddress

	for _, output := range outputs {
		addressesOuts := output.ScriptPubKey.Addresses
		outputSum += output.Value

		txOutputs = append(txOutputs, MultyAddress{addressesOuts, output.Value})
	}
	memPoolTx.outputs = txOutputs

	memPoolTx.amount = inputSum
	memPoolTx.fee = inputSum - outputSum

	memPoolTx.feeRate = int32(memPoolTx.fee / float64(memPoolTx.size) * 100000000)

	// log.Printf("\n **************************** Multy-New Tx Found *******************\n hash: %s, id: %s \n amount: %f , fee: %f , feeRate: %d \n Inputs: %v \n OutPuts: %v \n ****************************Multy-the best wallet*******************", memPoolTx.hash, memPoolTx.txid, memPoolTx.amount, memPoolTx.fee, memPoolTx.feeRate, memPoolTx.inputs, memPoolTx.outputs)
	// memPoolTx.hash, memPoolTx.txid, memPoolTx.amount, memPoolTx.fee, memPoolTx.feeRate, memPoolTx.inputs, memPoolTx.outputs

	var user store.User

	for _, input := range memPoolTx.inputs {
		for _, address := range input.address {
			usersData.Find(bson.M{"wallets.adresses.address": address}).One(&user)
			if user.Wallets != nil {
				chToClient <- CreateBtcTransactionWithUserID(user.UserID, txIn, "not implemented", memPoolTx.hash, input.amount)
				// add UserID related tx's to db
				rec := newTxInfo(txIn, memPoolTx.hash, address, input.amount)
				sel := bson.M{"userID": user.UserID}
				update := bson.M{"$push": bson.M{"transactions": rec}}
				err := usersData.Update(sel, update)
				if err != nil {
					fmt.Println(err)
				}
				// TODO: parse block
			}
			user = store.User{}
		}
	}

	for _, output := range memPoolTx.outputs {
		for _, address := range output.address {
			usersData.Find(bson.M{"wallets.adresses.address": address}).One(&user)
			if user.Wallets != nil {
				chToClient <- CreateBtcTransactionWithUserID(user.UserID, txOut, "not implemented", memPoolTx.hash, output.amount)
				// add UserID related tx's to db

				rec := newTxInfo(txOut, memPoolTx.hash, address, output.amount)
				sel := bson.M{"userID": user.UserID}
				update := bson.M{"$push": bson.M{"transactions": rec}}
				err := usersData.Update(sel, update)
				if err != nil {
					fmt.Println(err)
				}
				// TODO: parse block
			}
			user = store.User{}
		}
	}

	rec := newRecord(int(memPoolTx.feeRate), memPoolTx.hash)

	err := mempoolRates.Insert(rec)
	if err != nil {
		log.Println("ERR mempoolRates.Insert: ", err.Error())
		return err
	}

	//TODO save transaction as mem pool tx
	//TODO update fee rates table
	memPool = append(memPool, memPoolTx)

	log.Printf("[DEBUG] parseRawTransaction: new multy mempool; size=%d", len(memPool))
	return nil
}
func CreateBtcTransactionWithUserID(addr, userId, txType, txId string, amount float64) BtcTransactionWithUserID {
	return BtcTransactionWithUserID{
		UserID: userId,
		NotificationMsg: &BtcTransaction{
			TransactionType: txType,
			Amount:          amount,
			TxID:            txId,
			Address:         addr,
		},
	}
}

func newRecord(category int, hashTX string) Record {
	return Record{
		Category: category,
		HashTX:   hashTX,
	}
}

type Record struct {
	Category int    `json:"category"`
	HashTX   string `json:"hashTX"`
}

func newTxInfo(txType, txHash, address string, amount float64) TxInfo {
	return TxInfo{
		Type:    txType,
		TxHash:  txHash,
		Address: address,
		Amount:  amount,
		// timestamp
	}
}

type TxInfo struct {
	Type    string  `json:"type"`
	TxHash  string  `json:"txhash"`
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
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
		go findAndPushUserTransactions(blockTx)

		// !notify users that their transactions was applied in a block
		if err := usersData.Find(bson.M{"transactions.txhash": txHashStr}).One(&user); err != nil {
			log.Printf("[ERR] getAndParseNewBlock: usersData.Find: %s\n", err.Error())
			continue
		}

		if user.Wallets == nil {
			log.Println("[WARN] getAndParseNewBlock:  wallet is empty")
			continue
		}

		// 	// TODO: change slice to map
		var (
			output *store.BtcOutput
			ok     bool
		)
		for _, wallet := range user.Wallets {
			for _, addr := range wallet.Adresses {
				if output, ok = blockTx.Outputs[addr.Address]; !ok {
					continue
				}
				// got output with our address; notify user about it
				log.Println("[DEBUG] getAndParseNewBlock: address=", addr)
				chToClient <- CreateBtcTransactionWithUserID(addr.Address, user.UserID, txOut+" block", txHashStr, output.Amount)
			}
		}
	}
	log.Println("[DEBUG] hash iteration logic transactions done")

	// notify users that their transactions was applied in a block
	// for _, tx := range blockMSG.Transactions {
	// 	usersData.Find(bson.M{"transactions.txhash": tx.TxHash().String()}).One(&user)
	// 	if user.Wallets != nil {
	// 		if _, ok := user.Transactions[tx.TxHash().String()]; !ok {
	// 			log.Println("[DEBUG] getAndParseNewBlock: not our transaction")
	// 		}
	// 		addrs := user.Transactions[tx.TxHash().String()]
	// 		log.Printf("[DEBUG] transaction addrs: %+v\n", addrs)
	// 		chToClient <- CreateBtcTransactionWithUserID(user.UserID, userTx.Type+"block", tx.TxHash().String(), userTx.Amount)
	// 	}
	// 	user = store.User{}
	// }

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

func findAndPushUserTransactions(blockTx *store.BTCTransaction) {
	log.Print("[DEBUG] findAndPushUserTransactions")
	defer log.Print("[DEBUG] findAndPushUserTransactions done")

	user := store.User{}
	for _, output := range blockTx.Outputs {
		for _, address := range output.Address {
			log.Printf("[DEBUG] indAndPushUserTransactions: output.Addres=%s\n", address)
			usersData.Find(bson.M{"wallets.adresses.address": address}).One(&user)
			sel := bson.M{"userID": user.UserID}
			// ERROR 1
			update := bson.M{"$push": bson.M{"wallets.$.adresses.$.address.outputs": blockTx}}
			log.Printf("[DEBUG] indAndPushUserTransactions: sel=%+v/update=%+v\n", sel, update)
			err := usersData.Update(sel, update)
			if err != nil {
				fmt.Printf("[ERR] push transaction to db: %s\n", err.Error())
			}
		}
	}
	defer log.Print("[DEBUG] findAndPushUserTransactions done")
}

func parseBlockTransaction(txHash *chainhash.Hash) (*store.BTCTransaction, error) {
	log.Printf("[DEBUG] parseBlockTransaction()")
	currentRaw, err := rpcClient.GetRawTransactionVerbose(txHash)
	if err != nil {
		fmt.Printf("[ERR] parseBlockTransaction: GetRawTransactionVerbose(): %s\n", err.Error())
		return nil, err
	}

	blockTx := serealizeBTCTransaction(currentRaw)

	log.Printf("[DEBUG] parseBlockTransaction %+v\n", blockTx.Outputs)

	// log.Printf("\nMulty-New Tx Found hash: %s, id: %s \n amount: %f , fee: %f , feeRate: %d \n Inputs: %v \n OutPuts: %v \Multy-the best wallet*******************", memPoolTx.hash, memPoolTx.txid, memPoolTx.amount, memPoolTx.fee, memPoolTx.feeRate, memPoolTx.inputs, memPoolTx.outputs)
	log.Printf("[DEBUG] parseBlockTransaction() done")
	return blockTx, nil
}
