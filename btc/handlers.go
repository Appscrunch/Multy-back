package btc

/*
Copyright 2019 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/

import (
	"context"
	"io"
	"sync"
	"time"

	"gopkg.in/mgo.v2"

	"github.com/Appscrunch/Multy-back/currencies"
	pb "github.com/Appscrunch/Multy-back/node-streamer/btc"
	"github.com/Appscrunch/Multy-back/store"
	nsq "github.com/bitly/go-nsq"
	"gopkg.in/mgo.v2/bson"
)

func setGRPCHandlers(cli pb.NodeCommuunicationsClient, nsqProducer *nsq.Producer, networtkID int, wa chan pb.WatchAddress, mempool *map[string]int, m *sync.Mutex, resync *map[string]bool, resyncM *sync.Mutex) {

	mempoolCh := make(chan interface{})

	// Initial fill mempool respectively network id
	go func() {
		stream, err := cli.EventGetAllMempool(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventGetAllMempool: %s", err.Error())
		}

		for {
			mpRec, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("setGRPCHandlers: client.EventGetAllMempool: %s", err.Error())
			}

			mempoolCh <- store.MempoolRecord{
				Category: int(mpRec.Category),
				HashTx:   mpRec.HashTX,
			}

		}
	}()

	// Add transaction on every new tx on node
	go func() {
		stream, err := cli.EventAddMempoolRecord(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventAddMempoolRecord: %s", err.Error())
			// return nil, err
		}

		for {
			mpRec, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("setGRPCHandlers: client.EventAddMempoolRecord:stream.Recv: %s", err.Error())
			}

			mempoolCh <- store.MempoolRecord{
				Category: int(mpRec.Category),
				HashTx:   mpRec.HashTX,
			}

			if err != nil {
				log.Errorf("initGrpcClient: mpRates.Insert: %s", err.Error())
			}
		}
	}()

	// Deleting mempool record on block
	go func() {
		stream, err := cli.EventDeleteMempool(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventGetAllMempool: %s", err.Error())
			// return nil, err
		}

		for {
			mpRec, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("initGrpcClient: cli.EventDeleteMempool:stream.Recv: %s", err.Error())
			}

			mempoolCh <- mpRec.Hash

			if err != nil {
				log.Errorf("setGRPCHandlers:mpRates.Remove: %s", err.Error())
			} else {
				log.Debugf("Tx removed: %s", mpRec.Hash)
			}
		}

	}()

	// New spendable output
	go func() {
		stream, err := cli.EventAddSpendableOut(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventGetAllMempool: %s", err.Error())
		}

		spOutputs := &mgo.Collection{}
		spend := &mgo.Collection{}
		switch networtkID {
		case currencies.Main:
			spOutputs = spendableOutputs
			spend = spentOutputs
		case currencies.Test:
			spOutputs = spendableOutputsTest
			spend = spentOutputsTest
		default:
			log.Errorf("setGRPCHandlers: wrong networkID:")
		}

		for {
			gSpOut, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("initGrpcClient: cli.EventAddSpendableOut:stream.Recv: %s", err.Error())
			}

			query := bson.M{"userid": gSpOut.UserID, "txid": gSpOut.TxID, "address": gSpOut.Address}
			err = spend.Find(query).One(nil)

			if err == mgo.ErrNotFound {
				user := store.User{}
				sel := bson.M{"wallets.addresses.address": gSpOut.Address}
				err = usersData.Find(sel).One(&user)
				if err != nil && err != mgo.ErrNotFound {
					log.Errorf("SetWsHandlers: cli.On newIncomingTx: %s", err)
					return
				}
				spOut := generatedSpOutsToStore(gSpOut)

				log.Infof("Add spendable output : %v", gSpOut.String())

				exRates, err := GetLatestExchangeRate()
				if err != nil {
					log.Errorf("initGrpcClient: GetLatestExchangeRate: %s", err.Error())
				}
				spOut.StockExchangeRate = exRates

				query := bson.M{"userid": spOut.UserID, "txid": spOut.TxID, "address": spOut.Address}
				err = spOutputs.Find(query).One(nil)
				if err == mgo.ErrNotFound {
					// Insertion
					err := spOutputs.Insert(spOut)
					if err != nil {
						log.Errorf("Create spOutputs:txsData.Insert: %s", err.Error())
					}
					continue
				}
				if err != nil && err != mgo.ErrNotFound {
					log.Errorf("Create spOutputs:spOutputs.Find %s", err.Error())
					continue
				}

				update := bson.M{
					"$set": bson.M{
						"txstatus": spOut.TxStatus,
					},
				}
				err = spOutputs.Update(query, update)
				if err != nil {
					log.Errorf("CreateSpendableOutputs:spendableOutputs.Update: %s", err.Error())
				}
			}

		}

	}()

	// Delete spendable output
	go func() {
		stream, err := cli.EventDeleteSpendableOut(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventGetAllMempool: %s", err.Error())
		}
		spOutputs := &mgo.Collection{}
		spend := &mgo.Collection{}
		switch networtkID {
		case currencies.Main:
			spOutputs = spendableOutputs
			spend = spentOutputs
		case currencies.Test:
			spOutputs = spendableOutputsTest
			spend = spentOutputsTest
		default:
			log.Errorf("setGRPCHandlers: wrong networkID:")
		}
		for {
			del, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("initGrpcClient: cli.EventDeleteMempool:stream.Recv: %s", err.Error())
			}

			i := 0
			for {
				// Insert to spend collection
				err := spend.Insert(del)
				if err != nil {
					log.Errorf("DeleteSpendableOutputs:spend.Insert: %s", err)
				}
				query := bson.M{"userid": del.UserID, "txid": del.TxID, "address": del.Address}
				log.Infof("-------- query delete %v\n", query)
				err = spOutputs.Remove(query)
				if err != nil {
					log.Errorf("DeleteSpendableOutputs:spendableOutputs.Remove: %s", err.Error())
				} else {
					log.Infof("delete success √: %v", query)
					break
				}
				i++
				if i == 10 {
					break
				}
				time.Sleep(time.Second * 3)
			}
			log.Debugf("DeleteSpendableOutputs:spendableOutputs.Remove: %s", err)
		}
	}()

	// Add to transaction history record and send ws notification on tx
	go func() {
		stream, err := cli.NewTx(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventGetAllMempool: %s", err.Error())
		}

		for {
			gTx, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("initGrpcClient: cli.NewTx:stream.Recv: %s", err.Error())
			}
			tx := generatedTxDataToStore(gTx)

			setExchangeRates(&tx, gTx.Resync, tx.MempoolTime)
			setUserID(&tx)
			// setTxInfo(&tx)
			user := store.User{}
			// Set wallet index and address index in input
			for i := 0; i < len(tx.WalletsInput); i++ {
				sel := bson.M{"wallets.addresses.address": tx.WalletsInput[i].Address.Address}
				err := usersData.Find(sel).One(&user)
				if err == mgo.ErrNotFound {
					continue
				} else if err != nil && err != mgo.ErrNotFound {
					log.Errorf("initGrpcClient: cli.On newIncomingTx: %s", err)
				}
				for _, wallet := range user.Wallets {
					for _, addr := range wallet.Addresses {
						if addr.Address == tx.WalletsInput[i].Address.Address {
							tx.WalletsInput[i].WalletIndex = wallet.WalletIndex
							tx.WalletsInput[i].Address.AddressIndex = addr.AddressIndex
						}
					}
				}
			}

			for i := 0; i < len(tx.WalletsOutput); i++ {
				sel := bson.M{"wallets.addresses.address": tx.WalletsOutput[i].Address.Address}
				err := usersData.Find(sel).One(&user)
				if err == mgo.ErrNotFound {
					continue
				} else if err != nil && err != mgo.ErrNotFound {
					log.Errorf("initGrpcClient: cli.On newIncomingTx: %s", err)
				}

				for _, wallet := range user.Wallets {
					for _, addr := range wallet.Addresses {
						if addr.Address == tx.WalletsOutput[i].Address.Address {
							tx.WalletsOutput[i].WalletIndex = wallet.WalletIndex
							tx.WalletsOutput[i].Address.AddressIndex = addr.AddressIndex
						}
					}
				}
			}

			log.Infof("New tx history in- %v out-%v\n", tx.WalletsInput, tx.WalletsOutput)

			err = saveMultyTransaction(tx, networtkID, gTx.Resync)
			if err != nil {
				log.Errorf("initGrpcClient: saveMultyTransaction: %s", err)
			}
			updateWalletAndAddressDate(tx, networtkID)
			if !gTx.Resync {
				sendNotifyToClients(tx, nsqProducer, networtkID)
			}
		}
	}()

	// Resync tx history and spendable outputs
	go func() {
		spOutputs := &mgo.Collection{}
		spend := &mgo.Collection{}
		switch networtkID {
		case currencies.Main:
			spOutputs = spendableOutputs
			spend = spentOutputs
		case currencies.Test:
			spOutputs = spendableOutputsTest
			spend = spentOutputsTest
		default:
			log.Errorf("setGRPCHandlers: wrong networkID:")
		}

		stream, err := cli.ResyncAddress(context.Background(), &pb.Empty{})
		if err != nil {
			log.Errorf("setGRPCHandlers: cli.EventGetAllMempool: %s", err.Error())
		}

		for {
			rTxs, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Errorf("initGrpcClient: cli.NewTx:stream.Recv: %s", err.Error())
			}

			// Tx history
			for _, gTx := range rTxs.Txs {
				tx := generatedTxDataToStore(gTx)
				setExchangeRates(&tx, gTx.Resync, tx.MempoolTime)
				setUserID(&tx)
				user := store.User{}
				// Set wallet index and address index in input
				for i := 0; i < len(tx.WalletsInput); i++ {
					sel := bson.M{"wallets.addresses.address": tx.WalletsInput[i].Address.Address}
					err := usersData.Find(sel).One(&user)
					if err == mgo.ErrNotFound {
						continue
					} else if err != nil && err != mgo.ErrNotFound {
						log.Errorf("initGrpcClient: cli.On newIncomingTx: %s", err)
					}

					for _, wallet := range user.Wallets {
						for _, addr := range wallet.Addresses {
							if addr.Address == tx.WalletsInput[i].Address.Address {
								tx.WalletsInput[i].WalletIndex = wallet.WalletIndex
								tx.WalletsInput[i].Address.AddressIndex = addr.AddressIndex
							}
						}
					}
				}
				// Set wallet index and address index in output
				for i := 0; i < len(tx.WalletsOutput); i++ {
					sel := bson.M{"wallets.addresses.address": tx.WalletsOutput[i].Address.Address}
					err := usersData.Find(sel).One(&user)
					if err == mgo.ErrNotFound {
						continue
					} else if err != nil && err != mgo.ErrNotFound {
						log.Errorf("initGrpcClient: cli.On newIncomingTx: %s", err)
					}

					for _, wallet := range user.Wallets {
						for _, addr := range wallet.Addresses {
							if addr.Address == tx.WalletsOutput[i].Address.Address {
								tx.WalletsOutput[i].WalletIndex = wallet.WalletIndex
								tx.WalletsOutput[i].Address.AddressIndex = addr.AddressIndex
							}
						}
					}
				}
				err = saveMultyTransaction(tx, networtkID, gTx.Resync)
				if err != nil {
					log.Errorf("initGrpcClient: saveMultyTransaction: %s", err)
				}
				updateWalletAndAddressDate(tx, networtkID)
			}

			// SpOuts
			for _, gSpOut := range rTxs.SpOuts {
				query := bson.M{"userid": gSpOut.UserID, "txid": gSpOut.TxID, "address": gSpOut.Address}
				err = spend.Find(query).One(nil)
				if err == mgo.ErrNotFound {
					user := store.User{}
					sel := bson.M{"wallets.addresses.address": gSpOut.Address}
					err = usersData.Find(sel).One(&user)
					if err != nil && err != mgo.ErrNotFound {
						log.Errorf("SetWsHandlers: cli.On newIncomingTx: %s", err)
						return
					}
					spOut := generatedSpOutsToStore(gSpOut)
					log.Infof("Add spendable output : %v", gSpOut.String())
					exRates, err := GetLatestExchangeRate()
					if err != nil {
						log.Errorf("initGrpcClient: GetLatestExchangeRate: %s", err.Error())
					}
					spOut.StockExchangeRate = exRates

					query := bson.M{"userid": spOut.UserID, "txid": spOut.TxID, "address": spOut.Address}
					err = spOutputs.Find(query).One(nil)
					if err == mgo.ErrNotFound {
						// Insertion
						err := spOutputs.Insert(spOut)
						if err != nil {
							log.Errorf("Create spOutputs:txsData.Insert: %s", err.Error())
						}
						continue
					}
					if err != nil && err != mgo.ErrNotFound {
						log.Errorf("Create spOutputs:spOutputs.Find %s", err.Error())
						continue
					}
					update := bson.M{
						"$set": bson.M{
							"txstatus": spOut.TxStatus,
						},
					}
					err = spOutputs.Update(query, update)
					if err != nil {
						log.Errorf("CreateSpendableOutputs:spendableOutputs.Update: %s", err.Error())
					}
				}
			}

			// Del SpOuts
			for _, del := range rTxs.SpOutDelete {
				i := 0
				for {
					// Insert to spend collection
					err = spend.Insert(del)
					if err != nil {
						log.Errorf("DeleteSpendableOutputs:spend.Insert: %s", err)
					}
					query := bson.M{"userid": del.UserID, "txid": del.TxID, "address": del.Address}
					log.Infof("-------- query delete %v\n", query)
					err = spOutputs.Remove(query)
					if err != nil {
						log.Errorf("DeleteSpendableOutputs:spendableOutputs.Remove: %s", err.Error())
					} else {
						log.Infof("delete success √: %v", query)
						break
					}
					i++
					if i == 10 {
						break
					}
					time.Sleep(time.Second * 3)
				}
			}
			if len(rTxs.Txs) > 0 {

				resyncM.Lock()
				re := *resync
				delete(re, rTxs.Txs[0].TxAddress[0])
				// re[rTxs.SpOuts[0].Address] = false
				*resync = re
				resyncM.Unlock()
			}

		}

	}()

	// Watch for channel and push to node
	go func() {
		for {
			select {
			case addr := <-wa:
				a := addr
				rp, err := cli.EventAddNewAddress(context.Background(), &a)
				if err != nil {
					log.Errorf("NewAddressNode: cli.EventAddNewAddress %s\n", err.Error())
				}
				log.Debugf("EventAddNewAddress Reply %s", rp)

				rp, err = cli.EventResyncAddress(context.Background(), &pb.AddressToResync{
					Address:      addr.GetAddress(),
					UserID:       addr.GetUserID(),
					WalletIndex:  addr.GetWalletIndex(),
					AddressIndex: addr.GetWalletIndex(),
				})
				if err != nil {
					log.Errorf("EventResyncAddress: cli.EventResyncAddress %s\n", err.Error())
				}
				log.Debugf("EventResyncAddress Reply %s", rp)

			}
		}
	}()

	go func() {

		for {
			switch v := (<-mempoolCh).(type) {
			// default:
			// 	log.Errorf("Not found type: %v", v)
			case string:
				// Delete tx from pool
				newMap := *mempool
				delete(newMap, v)
				m.Lock()
				*mempool = newMap
				m.Unlock()
			case store.MempoolRecord:
				// Add tx to pool
				newMap := *mempool
				newMap[v.HashTx] = v.Category
				m.Lock()
				*mempool = newMap
				m.Unlock()
			}
		}
	}()

}
