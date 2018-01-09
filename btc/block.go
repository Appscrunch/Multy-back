package btc

import (
	"encoding/json"
	"fmt"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/Appscrunch/Multy-back/store"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func notifyNewBlockTx(hash *chainhash.Hash) {
	log.Debugf("New block connected %s", hash.String())

	// block Height
	// blockVerbose, err := rpcClient.GetBlockVerbose(hash)
	// blockHeight := blockVerbose.Height

	//parse all block transactions
	rawBlock, err := rpcClient.GetBlock(hash)
	allBlockTransactions, err := rawBlock.TxHashes()
	if err != nil {
		log.Errorf("parseNewBlock:rawBlock.TxHashes: %s", err.Error())
	}

	var user store.User

	// range over all block txID's and notify clients about including their transaction in block as input or output
	// delete by transaction hash record from mempool db to estimete tx speed
	for _, txHash := range allBlockTransactions {

		blockTxVerbose, err := rpcClient.GetRawTransactionVerbose(&txHash)
		if err != nil {
			log.Errorf("parseNewBlock:rpcClient.GetRawTransactionVerbose: %s", err.Error())
			continue
		}

		// delete all block transations from memPoolDB
		query := bson.M{"hashtx": blockTxVerbose.Txid}
		err = mempoolRates.Remove(query)
		if err != nil {
			log.Errorf("parseNewBlock:mempoolRates.Remove: %s", err.Error())
		} else {
			log.Debugf("Tx removed: %s", blockTxVerbose.Txid)
		}

		// parse block tx outputs and notify
		for _, out := range blockTxVerbose.Vout {
			for _, address := range out.ScriptPubKey.Addresses {

				query := bson.M{"wallets.addresses.address": address}
				err := usersData.Find(query).One(&user)
				if err != nil {
					continue
				}
				log.Debugf("[IS OUR USER] parseNewBlock: usersData.Find = %s", address)

				txMsq := BtcTransactionWithUserID{
					UserID: user.UserID,
					NotificationMsg: &BtcTransaction{
						TransactionType: txInBlock,
						Amount:          out.Value,
						TxID:            blockTxVerbose.Txid,
						Address:         address,
					},
				}
				sendNotifyToClients(&txMsq)

			}
		}

		// parse block tx inputs and notify
		for _, input := range blockTxVerbose.Vin {
			txHash, err := chainhash.NewHashFromStr(input.Txid)
			if err != nil {
				log.Errorf("parseNewBlock: chainhash.NewHashFromStr = %s", err)
			}
			previousTx, err := rpcClient.GetRawTransactionVerbose(txHash)
			if err != nil {
				log.Errorf("parseNewBlock:rpcClient.GetRawTransactionVerbose: %s ", err.Error())
				continue
			}

			for _, out := range previousTx.Vout {
				for _, address := range out.ScriptPubKey.Addresses {
					query := bson.M{"wallets.addresses.address": address}
					err := usersData.Find(query).One(&user)
					if err != nil {
						continue
					}
					log.Debugf("[IS OUR USER]-AS-OUT parseMempoolTransaction: usersData.Find = %s", address)

					txMsq := BtcTransactionWithUserID{
						UserID: user.UserID,
						NotificationMsg: &BtcTransaction{
							TransactionType: txOutBlock,
							Amount:          out.Value,
							TxID:            blockTxVerbose.Txid,
							Address:         address,
						},
					}
					sendNotifyToClients(&txMsq)
				}
			}
		}

	}
}

func sendNotifyToClients(txMsq *BtcTransactionWithUserID) {
	newTxJSON, err := json.Marshal(txMsq)
	if err != nil {
		log.Errorf("sendNotifyToClients: [%+v] %s\n", txMsq, err.Error())
		return
	}

	err = nsqProducer.Publish(TopicTransaction, newTxJSON)
	if err != nil {
		log.Errorf("nsq publish new transaction: [%+v] %s\n", txMsq, err.Error())
		return
	}
	return
}

func blockTransactions(hash *chainhash.Hash) {
	log.Debugf("New block connected %s", hash.String())

	// block Height
	blockVerbose, err := rpcClient.GetBlockVerbose(hash)
	blockHeight := blockVerbose.Height

	//parse all block transactions
	rawBlock, err := rpcClient.GetBlock(hash)
	allBlockTransactions, err := rawBlock.TxHashes()
	if err != nil {
		log.Errorf("parseNewBlock:rawBlock.TxHashes: %s", err.Error())
	}

	for _, txHash := range allBlockTransactions {

		blockTxVerbose, err := rpcClient.GetRawTransactionVerbose(&txHash)
		if err != nil {
			log.Errorf("parseNewBlock:rpcClient.GetRawTransactionVerbose: %s", err.Error())
			continue
		}

		// apear as output
		err = parseOutput(blockTxVerbose, blockHeight, TxStatusInBlockConfirmed)
		if err != nil {
			log.Errorf("parseNewBlock:parseOutput: %s", err.Error())
		}

		// apear as input
		err = parseInput(blockTxVerbose, blockHeight, TxStatusAppearedInBlockOutcoming)
		if err != nil {
			log.Errorf("parseNewBlock:parseInput: %s", err.Error())
		}

	}
}

type MultyTX struct {
	TxID        string                      `json:"txid"`
	TxHash      string                      `json:"txhash"`
	TxOutScript string                      `json:"txoutscript"`
	TxAddress   string                      `json:"address"`
	TxStatus    string                      `json:"txstatus"`
	TxOutAmount float64                     `json:"txoutamount"`
	TxOutID     int                         `json:"txoutid"`
	WalletIndex int                         `json:"walletindex"`
	BlockTime   int64                       `json:"blocktime"`
	BlockHeight int64                       `json:"blockheight"`
	TxFee       int64                       `json:"txfee"`
	FiatPrice   []store.ExchangeRatesRecord `json:"stockexchangerate"`
	TxInputs    []AddresAmount              `json:"txinputs"`
	TxOutputs   []AddresAmount              `json:"txoutputs"`
}
type AddresAmount struct {
	Address string `json:"exchangename"`
	Amount  int64  `json:"fiatequivalent"`
}

type StockExchangeRate struct {
	ExchangeName   string `json:"exchangename"`
	FiatEquivalent int    `json:"fiatequivalent"`
	TotalAmount    int    `json:"totalamount"`
}

type TxRecord struct {
	UserID       string    `json:"userid"`
	Transactions []MultyTX `json:"transactions"`
}

func newEmptyTx(userID string) TxRecord {
	return TxRecord{
		UserID:       userID,
		Transactions: []MultyTX{},
	}
}
func newAddresAmount(address string, amount int64) AddresAmount {
	return AddresAmount{
		Address: address,
		Amount:  amount,
	}
}

func newMultyTX(txID, txHash, txOutScript, txAddress, txStatus string, txOutAmount float64, txOutID, walletindex int, blockTime, blockHeight, fee int64, fiatPrice []store.ExchangeRatesRecord, inputs, outputs []AddresAmount) MultyTX {
	return MultyTX{
		TxID:        txID,
		TxHash:      txHash,
		TxOutScript: txOutScript,
		TxAddress:   txAddress,
		TxStatus:    txStatus,
		TxOutAmount: txOutAmount,
		TxOutID:     txOutID,
		WalletIndex: walletindex,
		BlockTime:   blockTime,
		BlockHeight: blockHeight,
		TxFee:       fee,
		FiatPrice:   fiatPrice,
		TxInputs:    inputs,
		TxOutputs:   outputs,
	}
}

const (
	TxStatusAppearedInMempoolIncoming = "incoming in mempool"
	TxStatusAppearedInBlockIncoming   = "incoming in block"

	TxStatusAppearedInMempoolOutcoming = "spend in mempool"
	TxStatusAppearedInBlockOutcoming   = "spend in block"

	TxStatusInBlockConfirmed = "in block confirmed"

	TxStatusRejectedFromBlock = "rejected block"
)

const (
	SixBlockConfirmation     = 6
	SixPlusBlockConfirmation = 7
)

func blockConfirmations(hash *chainhash.Hash) {
	blockVerbose, err := rpcClient.GetBlockVerbose(hash)
	blockHeight := blockVerbose.Height

	sel := bson.M{"transactions.txblockheight": bson.M{"$lte": blockHeight - SixBlockConfirmation, "$gte": blockHeight - SixPlusBlockConfirmation}}
	update := bson.M{
		"$set": bson.M{
			"transactions.$.txstatus": TxStatusInBlockConfirmed,
		},
	}
	err = txsData.Update(sel, update)
	if err != nil {
		log.Errorf("blockConfirmations:txsData.Update: %s", err.Error())
	}

	query := bson.M{"transactions.txblockheight": blockHeight + SixBlockConfirmation}

	var records []TxRecord
	txsData.Find(query).All(&records)
	for _, usertxs := range records {

		txMsq := BtcTransactionWithUserID{
			UserID: usertxs.UserID,
			NotificationMsg: &BtcTransaction{
				TransactionType: TxStatusInBlockConfirmed,
			},
		}
		sendNotifyToClients(&txMsq)
	}

}

func txInfo(txVerbose *btcjson.TxRawResult) ([]AddresAmount, []AddresAmount, int64, error) {

	inputs := []AddresAmount{}
	outputs := []AddresAmount{}
	var inputSum float64
	var outputSum float64

	for _, out := range txVerbose.Vout {
		for _, address := range out.ScriptPubKey.Addresses {
			amount := int64(out.Value * 100000000)
			outputs = append(outputs, newAddresAmount(address, amount))
		}
		outputSum += out.Value
	}
	for _, input := range txVerbose.Vin {
		hash, err := chainhash.NewHashFromStr(input.Txid)
		if err != nil {
			log.Errorf("txInfo:chainhash.NewHashFromStr: %s", err.Error())
			return nil, nil, 0, err
		}
		previousTxVerbose, err := rpcClient.GetRawTransactionVerbose(hash)
		if err != nil {
			log.Errorf("txInfo:rpcClient.GetRawTransactionVerbose: %s", err.Error())
			return nil, nil, 0, err
		}

		for _, address := range previousTxVerbose.Vout[input.Vout].ScriptPubKey.Addresses {
			amount := int64(previousTxVerbose.Vout[input.Vout].Value * 100000000)
			inputs = append(inputs, newAddresAmount(address, amount))
		}
		inputSum += previousTxVerbose.Vout[input.Vout].Value
	}
	fee := int64((inputSum - outputSum) * 100000000)

	return inputs, outputs, fee, nil
}

func GetLatestExchangeRate() ([]store.ExchangeRatesRecord, error) {
	selGdax := bson.M{
		"stockexchange": "Gdax",
	}
	selPoloniex := bson.M{
		"stockexchange": "Poloniex",
	}
	stocksGdax := store.ExchangeRatesRecord{}
	err := exRate.Find(selGdax).Sort("-timestamp").One(&stocksGdax)
	if err != nil {
		return nil, err
	}

	stocksPoloniex := store.ExchangeRatesRecord{}
	err = exRate.Find(selPoloniex).Sort("-timestamp").One(&stocksPoloniex)
	if err != nil {
		return nil, err
	}
	return []store.ExchangeRatesRecord{stocksPoloniex, stocksGdax}, nil

}
func fetchWalletIndex(wallets []store.Wallet, address string) int {
	var walletIndex int
	for _, wallet := range wallets {
		for _, addr := range wallet.Adresses {
			if addr.Address == address {
				walletIndex = wallet.WalletIndex
				break
			}
		}
	}
	return walletIndex
}

func parseOutput(txVerbose *btcjson.TxRawResult, blockHeight int64, txStatus string) error {
	user := store.User{}
	blockTimeUnixNano := time.Now().Unix()

	for _, output := range txVerbose.Vout {
		for _, address := range output.ScriptPubKey.Addresses {
			query := bson.M{"wallets.addresses.address": address}
			err := usersData.Find(query).One(&user)
			if err != nil {
				continue
				// is not our user
			}
			fmt.Println("[ITS OUR USER] ", user.UserID)

			walletIndex := fetchWalletIndex(user.Wallets, address)

			inputs, outputs, fee, err := txInfo(txVerbose)
			if err != nil {
				log.Errorf("parseInput:txInfo:output: %s", err.Error())
				continue
			}

			exRates, err := GetLatestExchangeRate()
			if err != nil {
				log.Errorf("parseOutput:GetLatestExchangeRate: %s", err.Error())
			}

			sel := bson.M{"userid": user.UserID, "transactions.txid": txVerbose.Txid, "transactions.txaddress": address}
			err = txsData.Find(sel).One(nil)
			if err == mgo.ErrNotFound {
				newTx := newMultyTX(txVerbose.Txid, txVerbose.Hash, output.ScriptPubKey.Hex, address, txStatus, output.Value, int(output.N), walletIndex, blockTimeUnixNano, blockHeight, fee, exRates, inputs, outputs)
				sel = bson.M{"userid": user.UserID}
				update := bson.M{"$push": bson.M{"transactions": newTx}}
				err = txsData.Update(sel, update)
				if err != nil {
					log.Errorf("parseInput.Update add new tx to user: %s", err.Error())
				}
				continue
			} else if err != nil && err != mgo.ErrNotFound {
				log.Errorf("parseInput:txsData.Find: %s", err.Error())
				continue
			}

			sel = bson.M{"userid": user.UserID, "transactions.txid": txVerbose.Txid, "transactions.txaddress": address}
			update := bson.M{
				"$set": bson.M{
					"transactions.$.txstatus":          txStatus,
					"transactions.$.txblockheight":     blockHeight,
					"transactions.$.txfee":             fee,
					"transactions.$.stockexchangerate": exRates,
					"transactions.$.txinputs":          inputs,
					"transactions.$.txoutputs":         outputs,
					"transactions.$.blocktime":         blockTimeUnixNano,
				},
			}

			err = txsData.Update(sel, update)
			if err != nil {
				log.Errorf("parseInput:outputsData.Insert case nil: %s", err.Error())
			}
		}
	}
	return nil
}

func parseInput(txVerbose *btcjson.TxRawResult, blockHeight int64, txStatus string) error {
	user := store.User{}
	blockTimeUnixNano := time.Now().Unix()

	for _, input := range txVerbose.Vin {

		previousTxVerbose, err := rawTxByTxid(input.Txid)
		if err != nil {
			log.Errorf("parseInput:rawTxByTxid: %s", err.Error())
			continue
		}

		for _, address := range previousTxVerbose.Vout[input.Vout].ScriptPubKey.Addresses {
			query := bson.M{"wallets.addresses.address": address}
			// Is it's our user transaction
			err := usersData.Find(query).One(&user)
			if err != nil {
				continue
				// Is not our user
			}

			log.Debugf("[ITS OUR USER] %s", user.UserID)

			inputs, outputs, fee, err := txInfo(txVerbose)
			if err != nil {
				log.Errorf("parseInput:txInfo:input: %s", err.Error())
				continue
			}
			exRates, err := GetLatestExchangeRate()
			if err != nil {
				log.Errorf("parseOutput:GetLatestExchangeRate: %s", err.Error())
			}

			walletIndex := fetchWalletIndex(user.Wallets, address)

			// Is our user already have this transactions.
			sel := bson.M{"userid": user.UserID, "transactions.txid": txVerbose.Txid, "transactions.txaddress": address}
			err = txsData.Find(sel).One(nil)
			if err == mgo.ErrNotFound {
				// User have no transaction like this. Add to DB.
				newTx := newMultyTX(txVerbose.Txid, txVerbose.Hash, previousTxVerbose.Vout[input.Vout].ScriptPubKey.Hex, address, txStatus, previousTxVerbose.Vout[input.Vout].Value, int(previousTxVerbose.Vout[input.Vout].N), walletIndex, blockTimeUnixNano, blockHeight, fee, exRates, inputs, outputs)
				sel = bson.M{"userid": user.UserID}
				update := bson.M{"$push": bson.M{"transactions": newTx}}
				err = txsData.Update(sel, update)
				if err != nil {
					log.Errorf("parseInput:txsData.Update add new tx to user: %s", err.Error())
				}
				continue
			} else if err != nil && err != mgo.ErrNotFound {
				log.Errorf("parseInput:txsData.Find: %s", err.Error())
				continue
			}

			// User have this transaction but with another status.
			// Update statsus, block height, exchange rate,block time, inputs and outputs.
			sel = bson.M{"userid": user.UserID, "transactions.txid": txVerbose.Txid, "transactions.txaddress": address}
			update := bson.M{
				"$set": bson.M{
					"transactions.$.txstatus":      txStatus,
					"transactions.$.txblockheight": blockHeight,
					"transactions.$.txinputs":      inputs,
					"transactions.$.txoutputs":     outputs,
					"transactions.$.blocktime":     blockTimeUnixNano,
				},
			}
			err = txsData.Update(sel, update)
			if err != nil {
				log.Errorf("parseInput:txsData.Update: %s", err.Error())
			}
		}
	}
	return nil
}

func rawTxByTxid(txid string) (*btcjson.TxRawResult, error) {
	hash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		return nil, err
	}
	previousTxVerbose, err := rpcClient.GetRawTransactionVerbose(hash)
	if err != nil {
		return nil, err
	}
	return previousTxVerbose, nil
}
