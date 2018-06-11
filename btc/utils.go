package btc

/*
Copyright 2018 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/Appscrunch/Multy-back/currencies"
	btcpb "github.com/Appscrunch/Multy-back/node-streamer/btc"
	"github.com/Appscrunch/Multy-back/store"
	nsq "github.com/bitly/go-nsq"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	exRate    *mgo.Collection
	usersData *mgo.Collection

	txsData          *mgo.Collection
	spendableOutputs *mgo.Collection
	spentOutputs     *mgo.Collection

	txsDataTest          *mgo.Collection
	spendableOutputsTest *mgo.Collection
	spentOutputsTest     *mgo.Collection
)

func updateWalletAndAddressDate(tx store.MultyTx, networkID int) error {

	for _, walletOutput := range tx.WalletsOutput {

		// update addresses last action time
		sel := bson.M{"userID": walletOutput.UserID, "wallets.addresses.address": walletOutput.Address.Address}
		update := bson.M{
			"$set": bson.M{
				"wallets.$.addresses.$[].lastActionTime": time.Now().Unix(),
			},
		}
		err := usersData.Update(sel, update)
		if err != nil {
			return errors.New("updateWalletAndAddressDate:usersData.Update: " + err.Error())
		}

		// TODO: fix "wallets.$.status":store.WalletStatusOK,
		// update wallets last action time
		// Set status to OK if some money transfered to this address
		user := store.User{}
		sel = bson.M{"userID": walletOutput.UserID, "wallets.walletIndex": walletOutput.WalletIndex, "wallets.addresses.address": walletOutput.Address.Address, "wallets.networkID": networkID, "wallets.currencyID": currencies.Bitcoin}
		err = usersData.Find(sel).One(&user)
		if err != nil {
			return errors.New("updateWalletAndAddressDate:usersData.Update: " + err.Error())
		}

		// TODO: fix hardcode if wallet.NetworkID == networkID && walletOutput.WalletIndex == walletindex && currencies.Bitcoin == currencyID {
		var flag bool
		var position int
		for i, wallet := range user.Wallets {
			if wallet.NetworkID == networkID && wallet.WalletIndex == walletOutput.WalletIndex && wallet.CurrencyID == currencies.Bitcoin {
				position = i
				flag = true
				break
			}
		}

		if flag {
			update = bson.M{
				"$set": bson.M{
					"wallets." + strconv.Itoa(position) + ".status":         store.WalletStatusOK,
					"wallets." + strconv.Itoa(position) + ".lastActionTime": time.Now().Unix(),
				},
			}
			err = usersData.Update(sel, update)
			if err != nil {
				return errors.New("updateWalletAndAddressDate:usersData.Update: " + err.Error())
			}

		}

	}

	for _, walletInput := range tx.WalletsInput {
		// update addresses last action time
		sel := bson.M{"userID": walletInput.UserID, "wallets.addresses.address": walletInput.Address.Address}
		update := bson.M{
			"$set": bson.M{
				"wallets.$.addresses.$[].lastActionTime": time.Now().Unix(),
			},
		}
		err := usersData.Update(sel, update)
		if err != nil {
			return errors.New("updateWalletAndAddressDate:usersData.Update: " + err.Error())
		}

		// update wallets last action time
		sel = bson.M{"userID": walletInput.UserID, "wallets.walletIndex": walletInput.WalletIndex, "wallets.addresses.address": walletInput.Address.Address}
		update = bson.M{
			"$set": bson.M{
				"wallets.$.lastActionTime": time.Now().Unix(),
			},
		}
		err = usersData.Update(sel, update)
		if err != nil {
			return errors.New("updateWalletAndAddressDate:usersData.Update: " + err.Error())
		}
	}

	return nil
}

// GetReSyncExchangeRate returns resynced exchange rates
func GetReSyncExchangeRate(time int64) ([]store.ExchangeRatesRecord, error) {
	selCCCAGG := bson.M{
		"stockexchange": "CCCAGG",
		"timestamp":     bson.M{"$lt": time},
	}
	stocksCCCAGG := store.ExchangeRatesRecord{}
	err := exRate.Find(selCCCAGG).Sort("-timestamp").One(&stocksCCCAGG)
	if err != nil {
		return nil, err
	}
	return []store.ExchangeRatesRecord{stocksCCCAGG}, nil
}

// GetLatestExchangeRate returns latest exchange rates
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

// setExchangeRates sets exchagne rates
func setExchangeRates(tx *store.MultyTx, isReSync bool, TxTime int64) {
	var err error
	if isReSync {
		rates, err := GetReSyncExchangeRate(TxTime)
		if err != nil {
			log.Errorf("processTransaction:ExchangeRates: %s", err.Error())
		}
		tx.StockExchangeRate = rates
		return
	}
	if !isReSync || err != nil {
		rates, err := GetLatestExchangeRate()
		if err != nil {
			log.Errorf("processTransaction:ExchangeRates: %s", err.Error())
		}
		tx.StockExchangeRate = rates
	}
}

// sendNotifyToClients sends notifications to clients
func sendNotifyToClients(tx store.MultyTx, nsqProducer *nsq.Producer, netid int) {

	for _, walletOutput := range tx.WalletsOutput {
		txMsq := store.TransactionWithUserID{
			UserID: walletOutput.UserID,
			NotificationMsg: &store.WsTxNotify{
				CurrencyID:      currencies.Bitcoin,
				NetworkID:       netid,
				Address:         walletOutput.Address.Address,
				Amount:          strconv.Itoa(int(tx.TxOutAmount)),
				TxID:            tx.TxID,
				TransactionType: tx.TxStatus,
			},
		}
		sendNotify(&txMsq, nsqProducer)
	}

	for _, walletInput := range tx.WalletsInput {
		txMsq := store.TransactionWithUserID{
			UserID: walletInput.UserID,
			NotificationMsg: &store.WsTxNotify{
				CurrencyID:      currencies.Bitcoin,
				NetworkID:       netid,
				Address:         walletInput.Address.Address,
				Amount:          strconv.Itoa(int(tx.TxOutAmount)),
				TxID:            tx.TxID,
				TransactionType: tx.TxStatus,
			},
		}

		sendNotify(&txMsq, nsqProducer)
	}
}

// sendNotify is a main method for sending notifications
func sendNotify(txMsq *store.TransactionWithUserID, nsqProducer *nsq.Producer) {
	newTxJSON, err := json.Marshal(txMsq)
	if err != nil {
		log.Errorf("sendNotifyToClients: [%+v] %s\n", txMsq, err.Error())
		return
	}

	err = nsqProducer.Publish(store.TopicTransaction, newTxJSON)
	if err != nil {
		log.Errorf("nsq publish new transaction: [%+v] %s\n", txMsq, err.Error())
		return
	}

	return
}

func generatedTxDataToStore(gSpOut *btcpb.BTCTransaction) store.MultyTx {
	outs := []store.AddressAmount{}
	for _, output := range gSpOut.TxOutputs {
		outs = append(outs, store.AddressAmount{
			Address: output.Address,
			Amount:  output.Amount,
		})
	}

	ins := []store.AddressAmount{}
	for _, inputs := range gSpOut.TxInputs {
		ins = append(ins, store.AddressAmount{
			Address: inputs.Address,
			Amount:  inputs.Amount,
		})
	}

	wInputs := []store.WalletForTx{}
	for _, walletOutputs := range gSpOut.WalletsOutput {
		wInputs = append(wInputs, store.WalletForTx{
			UserID: walletOutputs.Userid,
			Address: store.AddressForWallet{
				Address:         walletOutputs.Address,
				Amount:          walletOutputs.Amount,
				AddressOutIndex: int(walletOutputs.TxOutIndex),
			},
		})
	}

	wOutputs := []store.WalletForTx{}
	for _, walletInputs := range gSpOut.WalletsInput {
		wOutputs = append(wOutputs, store.WalletForTx{
			UserID: walletInputs.Userid,
			Address: store.AddressForWallet{
				Address:         walletInputs.Address,
				Amount:          walletInputs.Amount,
				AddressOutIndex: int(walletInputs.TxOutIndex),
			},
		})
	}

	return store.MultyTx{
		UserID:        gSpOut.UserID,
		TxID:          gSpOut.TxID,
		TxHash:        gSpOut.TxHash,
		TxOutScript:   gSpOut.TxOutScript,
		TxAddress:     gSpOut.TxAddress,
		TxStatus:      int(gSpOut.TxStatus),
		TxOutAmount:   gSpOut.TxOutAmount,
		BlockTime:     gSpOut.BlockTime,
		BlockHeight:   gSpOut.BlockHeight,
		Confirmations: int(gSpOut.Confirmations),
		TxFee:         gSpOut.TxFee,
		MempoolTime:   gSpOut.MempoolTime,
		TxInputs:      ins,
		TxOutputs:     outs,
		WalletsInput:  wInputs,
		WalletsOutput: wOutputs,
	}
}

func generatedSpOutsToStore(gSpOut *btcpb.AddSpOut) store.SpendableOutputs {
	return store.SpendableOutputs{
		TxID:         gSpOut.TxID,
		TxOutID:      int(gSpOut.TxOutID),
		TxOutAmount:  gSpOut.TxOutAmount,
		TxOutScript:  gSpOut.TxOutScript,
		Address:      gSpOut.Address,
		UserID:       gSpOut.UserID,
		TxStatus:     int(gSpOut.TxStatus),
		WalletIndex:  int(gSpOut.WalletIndex),
		AddressIndex: int(gSpOut.AddressIndex),
	}
}

func saveMultyTransaction(tx store.MultyTx, networtkID int, resync bool) error {

	txStore := &mgo.Collection{}
	switch networtkID {
	case currencies.Main:
		txStore = txsData
	case currencies.Test:
		txStore = txsDataTest
	default:
		return errors.New("saveMultyTransaction: wrong networkID")
	}

	// fetchedTxs := []store.MultyTx{}
	// query := bson.M{"txid": tx.TxID}
	// txStore.Find(query).All(&fetchedTxs)

	// Doubling txs fix on -1
	if tx.BlockHeight == -1 {
		multyTX := store.MultyTx{}
		if len(tx.WalletsInput) != 0 {
			sel := bson.M{"userid": tx.WalletsInput[0].UserID, "txid": tx.TxID, "walletsoutput.walletindex": tx.WalletsInput[0].WalletIndex}
			err := txStore.Find(sel).One(multyTX)
			if err == nil && multyTX.BlockHeight > -1 {
				return nil
			}
		}
		if len(tx.WalletsOutput) > 0 {
			sel := bson.M{"userid": tx.WalletsOutput[0].UserID, "txid": tx.TxID, "walletsoutput.walletindex": tx.WalletsOutput[0].WalletIndex}
			err := txStore.Find(sel).One(multyTX)
			if err == nil && multyTX.BlockHeight > -1 {
				return nil
			}
		}
	}

	// Doubling txs fix on a asynchronous err
	if len(tx.WalletsInput) != 0 {
		sel := bson.M{"userid": tx.UserID, "txid": tx.TxID, "walletsoutput.walletindex": tx.WalletsInput[0].WalletIndex, "mempooltime": tx.MempoolTime}
		err := txStore.Find(sel).One(nil)
		if err == nil {
			return nil
		}
	}
	if len(tx.WalletsOutput) > 0 {
		sel := bson.M{"userid": tx.UserID, "txid": tx.TxID, "walletsoutput.walletindex": tx.WalletsOutput[0].WalletIndex, "mempooltime": tx.MempoolTime}
		err := txStore.Find(sel).One(nil)
		if err == nil {
			return nil
		}
	}

	// This is splited transaction! That means that transaction's WalletsInputs and WalletsOutput have the same WalletIndex!
	//Here we have outgoing transaction for exact wallet!
	multyTX := store.MultyTx{}
	if tx.WalletsInput != nil && len(tx.WalletsInput) > 0 {
		// sel := bson.M{"userid": tx.WalletsInput[0].UserID, "transactions.txid": tx.TxID, "transactions.walletsinput.walletindex": tx.WalletsInput[0].WalletIndex}
		// sel := bson.M{"userid": tx.WalletsInput[0].UserID, "txid": tx.TxID, "walletsinput.walletindex": tx.WalletsInput[0].WalletIndex} // last
		sel := bson.M{"userid": tx.WalletsInput[0].UserID, "txid": tx.TxID, "walletsoutput.walletindex": tx.WalletsInput[0].WalletIndex}
		if tx.BlockHeight != -1 {
			sel = bson.M{"userid": tx.WalletsInput[0].UserID, "txid": tx.TxID, "walletsinput.walletindex": tx.WalletsInput[0].WalletIndex}
		}

		err := txStore.Find(sel).One(&multyTX)
		if err == mgo.ErrNotFound {
			// initial insertion
			err := txStore.Insert(tx)
			return err
		}
		if err != nil && err != mgo.ErrNotFound {
			// database error
			return err
		}

		update := bson.M{
			"$set": bson.M{
				"txstatus":      tx.TxStatus,
				"blockheight":   tx.BlockHeight,
				"confirmations": tx.Confirmations,
				"blocktime":     tx.BlockTime,
				"walletsoutput": tx.WalletsOutput,
				"walletsinput":  tx.WalletsInput,
			},
		}
		err = txStore.Update(sel, update)
		return err
	} else if tx.WalletsOutput != nil && len(tx.WalletsOutput) > 0 {
		// sel := bson.M{"userid": tx.WalletsOutput[0].UserID, "transactions.txid": tx.TxID, "transactions.walletsoutput.walletindex": tx.WalletsOutput[0].WalletIndex}
		sel := bson.M{"userid": tx.WalletsOutput[0].UserID, "txid": tx.TxID, "walletsoutput.walletindex": tx.WalletsOutput[0].WalletIndex}
		err := txStore.Find(sel).One(&multyTX)
		if err == mgo.ErrNotFound {
			// initial insertion
			err := txStore.Insert(tx)
			return err
		}
		if err != nil && err != mgo.ErrNotFound {
			// database error
			return err
		}

		update := bson.M{
			"$set": bson.M{
				"txstatus":      tx.TxStatus,
				"blockheight":   tx.BlockHeight,
				"confirmations": tx.Confirmations,
				"blocktime":     tx.BlockTime,
				"walletsoutput": tx.WalletsOutput,
				"walletsinput":  tx.WalletsInput,
			},
		}
		err = txStore.Update(sel, update)
		if err != nil {
			log.Errorf("saveMultyTransaction:txsData.Update %s", err.Error())
		}
		return err
	}
	return nil
}

func setUserID(tx *store.MultyTx) {
	user := store.User{}
	for _, address := range tx.TxAddress {
		query := bson.M{"wallets.addresses.address": address}
		err := usersData.Find(query).One(&user)
		if err != nil {
			log.Errorf("setUserID: usersData.Find: %s", err.Error())
		}
		tx.UserID = user.UserID
	}
}

func setTxInfo(tx *store.MultyTx) {
	user := store.User{}
	// set wallet index and address index in input
	for _, in := range tx.WalletsInput {
		sel := bson.M{"wallets.addresses.address": in.Address.Address}
		err := usersData.Find(sel).One(&user)
		if err == mgo.ErrNotFound {
			continue
		} else if err != nil && err != mgo.ErrNotFound {
			log.Errorf("initGrpcClient: cli.On newIncomingTx: %s", err)
		}

		for _, wallet := range user.Wallets {
			for i := 0; i < len(wallet.Addresses); i++ {
				if wallet.Addresses[i].Address == in.Address.Address {
					in.WalletIndex = wallet.WalletIndex
					in.Address.AddressIndex = wallet.Addresses[i].AddressIndex
				}
			}
		}
	}

	// set wallet index and address index in output
	for _, out := range tx.WalletsOutput {
		sel := bson.M{"wallets.addresses.address": out.Address.Address}
		err := usersData.Find(sel).One(&user)
		if err == mgo.ErrNotFound {
			continue
		} else if err != nil && err != mgo.ErrNotFound {
			log.Errorf("initGrpcClient: cli.On newIncomingTx: %s", err)
		}

		for _, wallet := range user.Wallets {
			for i := 0; i < len(wallet.Addresses); i++ {
				if wallet.Addresses[i].Address == out.Address.Address {
					out.WalletIndex = wallet.WalletIndex
					out.Address.AddressIndex = wallet.Addresses[i].AddressIndex
				}
			}
		}
	}
}
