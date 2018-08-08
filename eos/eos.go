/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package eos

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/Multy-io/Multy-EOS-node-service/proto"
	"github.com/Multy-io/Multy-back/currencies"
	"github.com/Multy-io/Multy-back/store"
	"github.com/bitly/go-nsq"
	"github.com/gin-gonic/gin/json"
	"github.com/jekabolt/slf"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var log = slf.WithContext("eos")

// Conn is EOS node connection handler
type Conn struct {
	Client         proto.NodeCommunicationsClient
	WatchAddresses chan proto.WatchAddress

	networkID int

	nsq *nsq.Producer

	restoreState *mgo.Collection
	txStore      *mgo.Collection
	exRate       *mgo.Collection
}

// NewConn creates all the connections
func NewConn(dbConf *store.Conf, grpcUrl string, nsqAddress string, txTable string, networkID int) (*Conn, error) {
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(nsqAddress, config)
	if err != nil {
		return nil, fmt.Errorf("nsq producer: %s", err)
	}

	dbConn, err := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    []string{dbConf.Address},
		Username: dbConf.Username,
		Password: dbConf.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("nsq producer: %s", err)
	}

	grpcConn, err := grpc.Dial(grpcUrl, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %s", err)
	}

	conn := &Conn{
		Client: proto.NewNodeCommunicationsClient(grpcConn),
		//TODO: chanel buffering?
		WatchAddresses: make(chan proto.WatchAddress),
		restoreState:   dbConn.DB(dbConf.DBRestoreState).C(dbConf.TableState),
		txStore:        dbConn.DB(dbConf.DBTx).C(txTable),
		exRate:         dbConn.DB(dbConf.DBStockExchangeRate).C("TableStockExchangeRate"),
		nsq:            producer,
		networkID:      networkID,
	}

	conn.runAsyncHandlers()

	return conn, nil
}

// runAsyncHandlers starts async events goroutines
func (conn *Conn) runAsyncHandlers() {
	go conn.watchAddressesHandler()

	go conn.newBlockHandler()

	go conn.newTxHandler()
}

// newBlockHandler processes block height info down to consumers
func (conn *Conn) newBlockHandler() {
	stream, err := conn.Client.NewBlock(context.TODO(), &proto.Empty{})
	if err != nil {
		log.Errorf("new block: %s", err)
	}
	for {
		newBlock, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Errorf("new block: %s", err)
		}

		err = conn.updateRestoreState(newBlock)
		if err != nil {
			log.Errorf("update restore state %s", err)
		}
	}
}

// watchAddressesHandler passes watched addresses to node service
func (conn *Conn) watchAddressesHandler() {
	for {
		addr := <-conn.WatchAddresses
		resp, err := conn.Client.AddNewAddress(context.TODO(), &addr)
		if err != nil {
			log.Errorf("EventAddNewAddress: %s", err)
		}
		log.Debugf("EventAddNewAddress Reply %s", resp)

		resp, err = conn.Client.ResyncAddress(context.TODO(), &proto.AddressToResync{
			Address: addr.GetAddress(),
		})
		if err != nil {
			log.Errorf("EventResyncAddress: cli.EventResyncAddress %s\n", err.Error())
		}
		log.Debugf("EventResyncAddress Reply %s", resp)
	}
}

// updateRestoreState updates db's restore state data
func (conn *Conn) updateRestoreState(height *proto.BlockHeight) error {
	query := bson.M{"currencyid": currencies.EOS, "networkid": conn.networkID}
	update := bson.M{
		"$set": bson.M{
			"blockheight": height.GetHeadBlockNum(),
		},
	}

	err := conn.restoreState.Update(query, update)
	if err == mgo.ErrNotFound {
		return conn.restoreState.Insert(store.LastState{
			BlockHeight: int64(height.GetHeadBlockNum()),
			CurrencyID:  currencies.EOS,
			NetworkID:   conn.networkID,
		})
	}

	return err
}

// newTxHandler processes NewTx stream down to consumers
func (conn *Conn) newTxHandler() {
	stream, err := conn.Client.NewTx(context.TODO(), &proto.Empty{})
	if err != nil {
		log.Errorf("new tx handler: %s", err)
		return
	}
	for {
		tx, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Errorf("new tx %s", err)
			continue
		}

		err = conn.saveActionRecord(tx)
		if err != nil {
			log.Errorf("eos save action: %s", err)
		}
		if !tx.GetResync() {
			err = conn.notifyClients(tx)
			if err != nil {
				log.Errorf("eos publish action: %s", err)
			}
		}
	}
}

// saveActionRecord updates db with action data
func (conn *Conn) saveActionRecord(action *proto.Action) error {
	log.Debugf("new action for %s", action.UserID)
	stored, err := conn.ActionToHistoryRecord(action)
	if err != nil {
		log.Errorf("save action: %s", err)
		return err
	}
	sel := bson.M{
		"user_id":        action.UserID,
		"wallet_index":   action.WalletIndex,
		"transaction_id": action.TransactionId,
		"action_index":   action.ActionIndex,
	}

	err = conn.txStore.Find(sel).One(stored)
	if err == mgo.ErrNotFound {
		err = conn.txStore.Insert(*stored)
	}
	// node service fetches new blocks
	// so no need for db update
	return err
}

func (conn *Conn) ActionToHistoryRecord(action *proto.Action) (*store.TransactionETH, error) {
	info, err := conn.Client.GetChainState(context.TODO(), &proto.Empty{})
	if err != nil {
		return nil, err
	}
	var status int
	if action.From == action.Address {
		if info.LastIrreversibleBlockNum >= action.BlockNum {
			status = store.TxStatusInBlockConfirmedOutcoming
		} else {
			status = store.TxStatusAppearedInBlockOutcoming
		}
	} else {
		if info.LastIrreversibleBlockNum >= action.BlockNum {
			status = store.TxStatusInBlockConfirmedIncoming
		} else {
			status = store.TxStatusAppearedInBlockIncoming
		}
	}
	if action.Amount.Symbol != "EOS" {
		return nil, errors.New("non EOS token transaction")
	}
	confirmations := int(info.LastIrreversibleBlockNum) - int(action.BlockNum)
	if confirmations < 0 {
		confirmations = 0
	}

	selBitfinex := bson.M{
		"stockexchange": "Bitfinex",
	}

	stocksBitfinex := store.ExchangeRatesRecord{}
	err = conn.exRate.Find(selBitfinex).Sort("-timestamp").One(&stocksBitfinex)

	tx := &store.TransactionETH{
		UserID:        action.UserID,
		AddressIndex:  int(action.AddressIndex),
		WalletIndex:   int(action.WalletIndex),
		BlockHeight:   int64(action.BlockNum),
		To:            action.To,
		From:          action.From,
		Status:        status,
		Confirmations: confirmations,
		Amount:        assetToString(action.Amount),
		BlockTime:     action.BlockTime,
		Hash:          hex.EncodeToString(action.TransactionId),
		StockExchangeRate: []store.ExchangeRatesRecord{
			store.ExchangeRatesRecord{
				Timestamp:     time.Now().Unix(),
				StockExchange: "Bitfinex",
				Exchanges:     stocksBitfinex.Exchanges,
			},
		},
	}

	// eth.SetExchangeRates(tx, action.Resync, action.BlockTime)

	return tx, nil
}

// assetToString presents Asset type as string
// based on eos-go Asset.String method
func assetToString(a *proto.Asset) string {
	return fmt.Sprintf("%d", a.Amount)
	//strInt := fmt.Sprintf("%d", a.Amount)
	//if len(strInt) < int(a.Precision+1) {
	//	// prepend `0` for the difference:
	//	strInt = strings.Repeat("0", int(a.Precision+uint32(1))-len(strInt)) + strInt
	//}
	//
	//var result string
	//if a.Precision == 0 {
	//	result = strInt
	//} else {
	//	result = strInt[:len(strInt)-int(a.Precision)] + "." + strInt[len(strInt)-int(a.Precision):]
	//}
	//
	//return fmt.Sprintf("%s %s", result, a.Symbol)
}

// notifyClients notifies clients about new EOS action
func (conn *Conn) notifyClients(action *proto.Action) error {
	msg := store.TransactionWithUserID{
		UserID: action.UserID,
		NotificationMsg: &store.WsTxNotify{
			WalletIndex:     int(action.WalletIndex),
			NetworkID:       conn.networkID,
			CurrencyID:      currencies.EOS,
			Address:         action.Address,
			TransactionType: int(action.Type),
			TxID:            fmt.Sprintf("%s:%d", hex.EncodeToString(action.TransactionId), action.ActionIndex),
			Amount:          assetToString(action.Amount),
			From:            action.From,
			To:              action.To,
		},
	}

	toSend, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.nsq.Publish(store.TopicTransaction, toSend)
}

// GetActions gets EOS actions for user's wallet from db
func (conn *Conn) GetActions(userID string, walletIndex, currencyID, networkID int) ([]store.TransactionETH, error) {
	var actions []store.TransactionETH
	err := conn.txStore.Find(bson.M{"userid": userID, "walletindex": walletIndex}).All(&actions)
	if err == mgo.ErrNotFound {
		log.Errorf("no eos transactions for userid %s", userID)
		return actions, err
	}
	return actions, err
}

// GetActionHistory gets history for users wallet
// and checks confirmations
func (conn *Conn) GetActionHistory(ctx context.Context, userID string, walletIndex, currencyID, networkID int) ([]store.TransactionETH, error) {
	state, err := conn.Client.GetChainState(ctx, &proto.Empty{})
	//TODO: get actions from node (needs to be implemented in eos-go)
	if err != nil {
		return nil, err
	}
	actions, err := conn.GetActions(userID, walletIndex, currencyID, networkID)
	if err != nil {
		return nil, err
	}

	for _, action := range actions {
		if int64(state.LastIrreversibleBlockNum) >= action.BlockHeight {
			if action.Status == store.TxStatusAppearedInBlockIncoming {
				action.Status = store.TxStatusInBlockConfirmedIncoming
			}
			if action.Status == store.TxStatusAppearedInBlockOutcoming {
				action.Status = store.TxStatusInBlockConfirmedOutcoming
			}
		}
		confirmations := int(state.LastIrreversibleBlockNum) - int(action.BlockHeight)
		if confirmations < 0 {
			confirmations = 0
		}
		action.Confirmations = confirmations
	}

	return actions, err
}