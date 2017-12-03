package store

import (
	"errors"
	"log"
)

var (
	errType        = errors.New("wrong database type")
	errEmplyConfig = errors.New("empty configuration for datastore")
)

const (
	storeMongo = "mongo"
	storeInMem = "inmemory"
)

const (
	tableUsers = "users"
)

type DataStore interface {
	//AddUser(cm *appuser.User) error
	//FindMember(id int) (appuser.User, error)
	Close() error
}

func Init(config map[string]interface{}) (DataStore, error) {
	log.Printf("[DEBUG] datastore config: %+v\n", config)
	if config == nil {
		return nil, errEmplyConfig
	}

	if _, ok := config["type"]; !ok {
		return nil, errType
	}

	switch config["type"].(string) {
	case "mongo":
		log.Println("[DEBUG] initializing mongo database")
		InitMongoDB(getMongoConfig(config))
	case "inmem":
		log.Println("[DEBUG] initializing inmemory datastore")
		return InitInMemoryStore()
	}
	return nil, errType
}
