package multyback

/*
Copyright 2018 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/

import (
	"context"
	"fmt"

	// exchanger "github.com/Appscrunch/Multy-back-exchange-service"
	"github.com/Appscrunch/Multy-back/btc"
	"github.com/Appscrunch/Multy-back/client"
	"github.com/Appscrunch/Multy-back/currencies"
	"github.com/Appscrunch/Multy-back/eth"
	btcpb "github.com/Appscrunch/Multy-back/node-streamer/btc"
	ethpb "github.com/Appscrunch/Multy-back/node-streamer/eth"
	"github.com/Appscrunch/Multy-back/store"
	"github.com/gin-gonic/gin"
	"github.com/jekabolt/slf"
)

var (
	log = slf.WithContext("multy-back")
)

const (
	defaultServerAddress = "0.0.0.0:6678"
	version              = "v1"
)

// Just constants
const (
	EventConnection    = "connection"
	EventInitialAdd    = "allUsers"
	EventResyncAddress = "resync"
	EventSendRawTx     = "sendRaw"
	EventAddNewAddress = "newUser"
	Room               = "node"
)

// Multy is a main struct of service
type Multy struct {
	config     *Configuration
	clientPool *client.SocketIOConnectedPool
	route      *gin.Engine

	userStore store.UserStore

	restClient     *client.RestClient
	firebaseClient *client.FirebaseClient

	BTC *btc.Conn
	ETH *eth.Conn
}

// Init initializes Multy instance
func Init(conf *Configuration) (*Multy, error) {
	multy := &Multy{
		config: conf,
	}
	// DB initialization
	userStore, err := store.InitUserStore(conf.Database)
	if err != nil {
		return nil, fmt.Errorf("DB initialization: %s on port %s", err.Error(), conf.Database.Address)
	}
	multy.userStore = userStore
	log.Infof("UserStore initialization done on %s √", conf.Database)

	// exchange rates
	// exchange := &exchanger.Exchanger{}
	// exchange.InitExchanger(conf.ExchangerConfiguration)

	//BTC
	btcCli, err := btc.InitHandlers(&conf.Database, conf.SupportedNodes, conf.NSQAddress)
	if err != nil {
		return nil, fmt.Errorf("Init: btc.InitHandlers: %s", err.Error())
	}
	btcVer, err := btcCli.CliMain.ServiceInfo(context.Background(), &btcpb.Empty{})
	multy.BTC = btcCli
	log.Infof(" BTC initialization done on %v √", btcVer)

	// ETH
	ethCli, err := eth.InitHandlers(&conf.Database, conf.SupportedNodes, conf.NSQAddress)
	if err != nil {
		return nil, fmt.Errorf("Init: btc.InitHandlers: %s", err.Error())
	}
	ethVer, err := ethCli.CliMain.ServiceInfo(context.Background(), &ethpb.Empty{})
	multy.ETH = ethCli
	log.Infof(" ETH initialization done on %v √", ethVer)

	//users data set
	sv, err := multy.SetUserData(multy.userStore, conf.SupportedNodes)
	if err != nil {
		return nil, fmt.Errorf("Init: multy.SetUserData: %s", err.Error())
	}
	log.Infof("Users data  initialization done √")

	log.Debugf("Server versions %v", sv)

	// REST handlers
	if err = multy.initHTTPRoutes(conf); err != nil {
		return nil, fmt.Errorf("Router initialization: %s", err.Error())
	}
	return multy, nil
}

// SetUserData make initial userdata to node service
func (multy *Multy) SetUserData(userStore store.UserStore, ct []store.CoinType) ([]store.ServiceInfo, error) {
	servicesInfo := []store.ServiceInfo{}
	for _, conCred := range ct {
		usersData, err := userStore.FindUserDataChain(conCred.СurrencyID, conCred.NetworkID)
		if err != nil {
			return servicesInfo, fmt.Errorf("SetUserData: userStore.FindUserDataChain: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
		}
		if len(usersData) == 0 {
			log.Infof("Empty userdata")
		}

		// rp, err = multy.CliTest.SyncState(context.Background(), &btcpb.BlockHeight{
		// 	Height: ls.BTCTestBlock,
		// })

		switch conCred.СurrencyID {
		case currencies.Bitcoin:
			var cli btcpb.NodeCommuunicationsClient
			switch conCred.NetworkID {
			case currencies.Main:
				cli = multy.BTC.CliMain
			case currencies.Test:
				cli = multy.BTC.CliTest
			default:
				log.Errorf("setGRPCHandlers: wrong networkID:")
			}

			//TODO: Re State
			// h, err := m.userStore.FethLastSyncBlockState(conCred.СurrencyID, conCred.NetworkID)
			// if err != nil {
			// 	log.Errorf("SetUserData:  btcCli.CliMain.cli.FethLastSyncBlockState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// 	// return servicesInfo, fmt.Errorf("SetUserData:  btcCli.CliMain.FethLastSyncBlockState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// }
			// rp, err := cli.SyncState(context.Background(), &btcpb.BlockHeight{
			// 	Height: h,
			// })
			// if err != nil {
			// 	log.Errorf("SetUserData:  btcCli.CliMain.cli.SyncState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// 	// return servicesInfo, fmt.Errorf("SetUserData:  btcCli.CliMain.cli.SyncState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// }

			// if strings.Contains("err:", rp.GetMessage()) {
			// 	log.Errorf("SetUserData:  Contains err : curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// 	// return servicesInfo, fmt.Errorf("SetUserData:  Contains err : curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// }
			// log.Errorf("BTC ++++++++++ %v", h)

			genUd := btcpb.UsersData{
				Map: map[string]*btcpb.AddressExtended{},
			}
			for address, ex := range usersData {
				genUd.Map[address] = &btcpb.AddressExtended{
					UserID:       ex.UserID,
					WalletIndex:  int32(ex.WalletIndex),
					AddressIndex: int32(ex.AddressIndex),
				}
			}
			resp, err := cli.EventInitialAdd(context.Background(), &genUd)
			if err != nil {
				return servicesInfo, fmt.Errorf("SetUserData:  btcCli.CliMain.EventInitialAdd: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			}
			log.Debugf("Btc EventInitialAdd: resp: %s", resp.Message)

			sv, err := cli.ServiceInfo(context.Background(), &btcpb.Empty{})
			if err != nil {
				return servicesInfo, fmt.Errorf("SetUserData:  cli.ServiceInfo: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			}
			servicesInfo = append(servicesInfo, store.ServiceInfo{
				Branch:    sv.Branch,
				Commit:    sv.Commit,
				Buildtime: sv.Buildtime,
				Lasttag:   sv.Lasttag,
			})

		case currencies.Ether:
			var cli ethpb.NodeCommuunicationsClient
			switch conCred.NetworkID {
			case currencies.ETHMain:
				cli = multy.ETH.CliMain
			case currencies.ETHTest:
				cli = multy.ETH.CliTest
			default:
				log.Errorf("setGRPCHandlers: wrong networkID:")
			}

			// TODO: Re State
			// h, err := m.userStore.FethLastSyncBlockState(conCred.СurrencyID, conCred.NetworkID)
			// if err != nil {
			// 	log.Errorf("SetUserData:  btcCli.CliMain.cli.FethLastSyncBlockState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// 	// return servicesInfo, fmt.Errorf("SetUserData:  btcCli.CliMain.FethLastSyncBlockState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// }
			// log.Errorf("ETH ++++++++++ %v", h)
			// rp, err := cli.SyncState(context.Background(), &ethpb.BlockHeight{
			// 	Height: h,
			// })
			// if err != nil {
			// 	log.Errorf("SetUserData:  btcCli.CliMain.cli.SyncState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// 	// return servicesInfo, fmt.Errorf("SetUserData:  btcCli.CliMain.cli.SyncState: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// }

			// if strings.Contains("err:", rp.GetMessage()) {
			// 	log.Errorf("SetUserData:  Contains err : curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// 	// return servicesInfo, fmt.Errorf("SetUserData:  Contains err : curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			// }

			genUd := ethpb.UsersData{
				Map: map[string]*ethpb.AddressExtended{},
			}

			for address, ex := range usersData {
				genUd.Map[address] = &ethpb.AddressExtended{
					UserID:       ex.UserID,
					WalletIndex:  int32(ex.WalletIndex),
					AddressIndex: int32(ex.AddressIndex),
				}
			}
			resp, err := cli.EventInitialAdd(context.Background(), &genUd)
			if err != nil {
				return servicesInfo, fmt.Errorf("SetUserData: Ether.EventInitialAdd: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			}
			log.Debugf("Ether cli.EventInitialAdd: resp: %s", resp.Message)

			sv, err := cli.ServiceInfo(context.Background(), &ethpb.Empty{})
			if err != nil {
				return servicesInfo, fmt.Errorf("SetUserData:  cli.ServiceInfo: curID :%d netID :%d err =%s", conCred.СurrencyID, conCred.NetworkID, err.Error())
			}
			servicesInfo = append(servicesInfo, store.ServiceInfo{
				Branch:    sv.Branch,
				Commit:    sv.Commit,
				Buildtime: sv.Buildtime,
				Lasttag:   sv.Lasttag,
			})
		}
	}

	return nil, nil
}

// initHTTPRoutes initialize client communication services
// - http
// - socketio
// - firebase
func (multy *Multy) initHTTPRoutes(conf *Configuration) error {
	router := gin.Default()
	multy.route = router
	//
	gin.SetMode(gin.DebugMode)

	restClient, err := client.SetRestHandlers(
		multy.userStore,
		router,
		conf.DonationAddresses,
		multy.BTC,
		multy.ETH,
		conf.MultyVerison,
		conf.Secretkey,
	)
	if err != nil {
		return err
	}
	multy.restClient = restClient

	// socketIO server initialization. server -> mobile client
	socketIORoute := router.Group("/socketio")
	socketIOPool, err := client.SetSocketIOHandlers(multy.restClient, multy.BTC, multy.ETH, socketIORoute, conf.SocketioAddr, conf.NSQAddress, multy.userStore)
	if err != nil {
		return err
	}
	multy.clientPool = socketIOPool

	firebaseClient, err := client.InitFirebaseConn(&conf.Firebase, multy.route, conf.NSQAddress)
	if err != nil {
		return err
	}
	multy.firebaseClient = firebaseClient

	return nil
}

// Run runs service
func (multy *Multy) Run() error {
	log.Info("Running server")
	multy.route.Run(multy.config.RestAddress)
	return nil
}

func fethCoinType(coinTypes []store.CoinType, currencyID, networkID int) (*store.CoinType, error) {
	for _, ct := range coinTypes {
		if ct.СurrencyID == currencyID && ct.NetworkID == networkID {
			return &ct, nil
		}
	}
	return nil, fmt.Errorf("fethCoinType: no such coin in config")
}
