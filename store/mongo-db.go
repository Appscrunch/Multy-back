package store

import (
	"github.com/Appscrunch/Multy-back/types"
	"gopkg.in/mgo.v2"
)

type mongoDB struct {
	*mgo.Session
	config *MongoConfig
}

//MongoConfig is a config for mongo database
type MongoConfig struct {
	Type     string
	User     string
	Password string
	NameDB   string
	Address  string
}

// MigrationConf configures database migration
// TODO: add migrations
type MigrationConf struct {
	Enable      bool
	Directory   string
	Version     int
	StatusTable string
}

func getMongoConfig(rawConf map[string]interface{}) *MongoConfig {
	return nil
}

func InitMongoDB(conf *MongoConfig) (DataStore, error) {
	if conf == nil {
		return nil, errEmplyConfig
	}
	session, err := mgo.Dial(conf.Address)
	if err != nil {
		return nil, err
	}
	return &mongoDB{Session: session}, nil
}

func (mDB *mongoDB) AddUser(user *types.User) error {
	session := mDB.Copy()
	// defer session.Close()
	// TODO: check if user exists
	users := session.DB(mDB.config.NameDB).C(tableUsers)
	return users.Insert(user)
}

func (mDB *mongoDB) Close() error {
	return mDB.Close()
}

func (mDB *mongoDB) FindMember(id int) (types.User, error) {
	// session := ms.Copy()
	// defer session.Close()
	// personnel := session.DB("kek").C("users")
	// cm := CrewMember{}
	// err := personnel.Find(bson.M{"id": id}).One(&cm)
	return types.User{}, nil
}
