package multyback

import (
	"fmt"
	"log"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/Appscrunch/Multy-back/client"
	"github.com/Appscrunch/Multy-back/store"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"
)

const (
	defaultServerAddress = "0.0.0.0:7778"
	version              = "v1"
)

// Multy is a main struct of service
type Multy struct {
	config *Configuration

	clientPool *client.SocketIOConnectedPool
	route      *gin.Engine

	userStore store.UserStore

	btcClient  *rpcclient.Client
	restClient *client.RestClient
}

// Init initializes Multy instance
func Init(conf *Configuration) (*Multy, error) {
	multy := &Multy{
		config: conf,
	}

	userStore, err := store.InitUserStore(&conf.Database)
	if err != nil {
		return nil, err
	}
	multy.userStore = userStore

	// TODO: add channels for communitation
	log.Println("[DEBUG] InitHandlers")
	btcClient, err := btc.InitHandlers(userStore, btcConnCfg)
	if err != nil {
		return nil, fmt.Errorf("blockchain api initialization: %s", err.Error())
	}
	log.Println("[INFO] btc handlers initialization done")
	multy.btcClient = btcClient

	if err = multy.initRoute(conf.SocketioAddr); err != nil {
		return nil, fmt.Errorf("router initialization: %s", err.Error())
	}

	log.Println("[DEBUG] init done")
	return multy, nil
}

func (multy *Multy) initRoute(address string) error {
	router := gin.Default()

	gin.SetMode(gin.DebugMode)

	socketIORoute := router.Group("/socketio")
	socketIOPool, err := client.SetSocketIOHandlers(socketIORoute)
	if err != nil {
		return err
	}

	multy.route = router
	multy.clientPool = socketIOPool

	restClient, err := client.SetRestHandlers(
		multy.userStore,
		multy.config.BTCAPITest,
		multy.config.BTCAPIMain,
		router, multy.btcClient)
	if err != nil {
		return err
	}
	multy.restClient = restClient

	return nil
}

// Run runs service
func (multy *Multy) Run() error {
	if multy.config.RestAddress == "" {
		log.Println("[INFO] listening on default addres: ", defaultServerAddress)
	}
	multy.config.RestAddress = defaultServerAddress

	log.Println("[DEBUG] running server")
	multy.route.Run(multy.config.RestAddress)
	return nil
}
