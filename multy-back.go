package multyback

import (
	"fmt"
	"log"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/Appscrunch/Multy-back/store"
)

// Multy is a main struct of service
type Multy struct {
	config           *Configuration
	connectedClients map[string][]*Client
	dataStore        store.DataStore
	memPool          store.DataStore
}

// Client is a struct with client data
type Client struct {
	id      int64
	token   string
	wsConns []string
}

// Init initializes Multy instance
func Init(conf *Configuration) (*Multy, error) {
	m := &Multy{
		config:           conf,
		connectedClients: make(map[string][]*Client, 0),
	}

	dataStore, err := store.Init(m.config.DataStore)
	if err != nil {
		return nil, fmt.Errorf("database initialization: %s", err.Error())
	}
	m.dataStore = dataStore

	log.Println("[DEBUG] InitHandlers")
	err = btc.InitHandlers()
	if err != nil {
		return nil, fmt.Errorf("blockchain api initialization: %s", err.Error())
	}

	log.Println("[DEBUG] init done")
	return m, nil
}

// Run runs service
func (m *Multy) Run() error {
	log.Println("Run: not implemented")

	return nil
}
