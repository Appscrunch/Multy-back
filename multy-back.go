package multyback

import (
	"fmt"
	"log"
	"time"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/Appscrunch/Multy-back/client"
	"github.com/Appscrunch/Multy-back/store"
	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"
)

const (
	defaultServerAddress = "localhost:7778"
	version              = "v1"
)

// Multy is a main struct of service
type Multy struct {
	config     *Configuration
	clientPool *client.ConnectedPool
	dataStore  store.DataStore
	memPool    store.DataStore
	route      *gin.Engine
	socketIO   *socketio.Server
}

// Init initializes Multy instance
func Init(conf *Configuration) (*Multy, error) {
	m := &Multy{
		config:     conf,
		clientPool: client.InitConnectedPool(),
	}

	dataStore, err := store.Init(m.config.DataStore)
	if err != nil {
		return nil, fmt.Errorf("database initialization: %s", err.Error())
	}
	m.dataStore = dataStore

	// TODO: add channels for communitation
	log.Println("[DEBUG] InitHandlers")
	err = btc.InitHandlers()
	if err != nil {
		return nil, fmt.Errorf("blockchain api initialization: %s", err.Error())
	}

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
	err := client.SetRestHandlers(restRoute, m.clientPool)
	if err != nil {
		return err
	}

	return nil
}

// Run runs service
func (m *Multy) Run() error {
	if m.config.Address == "" {
		log.Println("[INFO] listening on default addres: ", defaultServerAddress)
	}
	m.config.Address = defaultServerAddress
	time.Sleep(time.Second * 1000)
	//m.route.Run(m.config.Address)
	return nil
}
