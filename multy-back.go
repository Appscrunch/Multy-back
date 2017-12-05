package multyback

import (
	"fmt"
	"log"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/Appscrunch/Multy-back/client"
	"github.com/Appscrunch/Multy-back/store"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"
)

const (
	defaultServerAddress = "0.0.0.0:8080"
	version              = "v1"
)

// Multy is a main struct of service
type Multy struct {
	config     *Configuration
	clientPool *client.ConnectedPool
	//dataStore  store.DataStore
	//memPool    store.DataStore
	userStore store.UserStore
	route     *gin.Engine

	socketIO   *socketio.Server
	rpcClient  *rpcclient.Client
	restClient *client.RestClient
}

// Init initializes Multy instance
func Init(conf *Configuration) (*Multy, error) {
	m := &Multy{
		config:     conf,
		clientPool: client.InitConnectedPool(),
	}

	userStore, err := store.InitUserStore(conf.DataStoreAddress)
	if err != nil {
		return nil, err
	}
	m.userStore = userStore

	// TODO: add channels for communitation
	log.Println("[DEBUG] InitHandlers")
	rpcClient, err := btc.InitHandlers()
	if err != nil {
		return nil, fmt.Errorf("blockchain api initialization: %s", err.Error())
	}
	m.rpcClient = rpcClient

	if err = m.initRoute(conf.Address); err != nil {
		return nil, fmt.Errorf("router initialization: %s", err.Error())
	}

	log.Println("[DEBUG] init done")
	return m, nil
}

func (m *Multy) initRoute(address string) error {
	router := gin.Default()
	rWithVersion := router.Group("/" + version)

	gin.SetMode(gin.DebugMode)

	socketIORoute := router.Group("/socketio")
	socketIOServer, err := client.SetSocketIOHandlers(socketIORoute, m.clientPool)
	if err != nil {
		return err
	}

	m.route = router
	m.socketIO = socketIOServer

	restRoute := rWithVersion.Group("/rest")
	restClient, err := client.SetRestHandlers(m.userStore, restRoute, m.rpcClient)
	if err != nil {
		return err
	}
	m.restClient = restClient

	return nil
}

// Run runs service
func (m *Multy) Run() error {
	if m.config.Address == "" {
		log.Println("[INFO] listening on default addres: ", defaultServerAddress)
	}
	m.config.Address = defaultServerAddress

	log.Println("[DEBUG] running server")
	m.route.Run(m.config.Address)
	return nil
}
