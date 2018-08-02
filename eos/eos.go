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
	"github.com/Multy-io/Multy-EOS-node-service/proto"
	"github.com/Multy-io/Multy-back/currencies"
	"github.com/Multy-io/Multy-back/store"
	"github.com/bitly/go-nsq"
	"github.com/gin-gonic/gin/json"
	"github.com/jekabolt/slf"
	"google.golang.org/grpc"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"strings"
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
}

// NewConn creates all the connections
func NewConn(dbConf *store.Conf, grpcUrl string, nsqAddress string, txTable string) (*Conn, error) {
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
		nsq:            producer,
		networkID:      currencies.Main,
	}

	conn.runAsyncHandlers()

	return conn, nil
}

// getGrpc creates grpc connection to the corresponding node service
func getGrpc(nodes []store.CoinType, currencyID, networkID int) (*grpc.ClientConn, error) {
	for _, ct := range nodes {
		if ct.СurrencyID == currencyID && ct.NetworkID == networkID {
			return grpc.Dial(ct.GRPCUrl, grpc.WithInsecure())
		}
	}
	return nil, fmt.Errorf("no such coin in config")
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
	sel := bson.M{
		"user_id":        action.UserID,
		"wallet_index":   action.WalletIndex,
		"transaction_id": action.TransactionId,
		"action_index":   action.ActionIndex,
	}

	var stored proto.Action
	err := conn.txStore.Find(sel).One(&stored)
	if err == mgo.ErrNotFound {
		err = conn.txStore.Insert(action)
	}
	// node service fetches new blocks
	// so no need for db update
	return err
}

// assetToString presents Asset type as string
// based on eos-go Asset.String method
func assetToString(a *proto.Asset) string {
	strInt := fmt.Sprintf("%d", a.Amount)
	if len(strInt) < int(a.Precision+1) {
		// prepend `0` for the difference:
		strInt = strings.Repeat("0", int(a.Precision+uint32(1))-len(strInt)) + strInt
	}

	var result string
	if a.Precision == 0 {
		result = strInt
	} else {
		result = strInt[:len(strInt)-int(a.Precision)] + "." + strInt[len(strInt)-int(a.Precision):]
	}

	return fmt.Sprintf("%s %s", result, a.Symbol)
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
func (conn *Conn) GetActions(userID string, walletIndex, currencyID, networkID int) ([]proto.Action, error) {
	var actions []proto.Action
	err := conn.txStore.Find(bson.M{"userid": userID, "walletindex": walletIndex}).All(&actions)
	if err == mgo.ErrNotFound {
		log.Errorf("no eos transactions for userid %s", userID)
		return actions, err
	}
	return actions, err
}

// GetActionHistory gets history for users wallet
// and checks confirmations
func (conn *Conn) GetActionHistory(ctx context.Context, userID string, walletIndex, currencyID, networkID int) ([]ActionHistoryRecord, error) {
	state, err := conn.Client.GetChainState(ctx, &proto.Empty{})
	//TODO: get actions from node (needs to be implemented in eos-go)
	if err != nil {
		return nil, err
	}
	actions, err := conn.GetActions(userID, walletIndex, currencyID, networkID)
	if err != nil {
		return nil, err
	}

	history := make([]ActionHistoryRecord, len(actions))
	for i := range actions {
		history[i].Action = actions[i]
		if state.LastIrreversibleBlockNum >= history[i].BlockNum { // action is in irreversible state
			history[i].Confirmations = 1
		} else {
			history[i].Confirmations = 0
		}
	}

	return history, err
}

// ActionHistoryRecord is a record that is pushed to client
// it is extended with confirmations data
type ActionHistoryRecord struct {
	proto.Action

	Confirmations int `json:"confirmations"`
}
