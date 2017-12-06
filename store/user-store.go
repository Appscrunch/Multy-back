package store

import (
	"errors"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	errType        = errors.New("wrong database type")
	errEmplyConfig = errors.New("empty configuration for datastore")
)

const (
	tableUsers = "userCollection"
	dbUsers    = "userDB"
)

type UserStore interface {
	//GetSession()
	GetUserByDevice(device bson.M, user *User)
	Update(sel, update bson.M) error
	Insert(user User) error
	Close() error
	FindUser(query bson.M, user *User) error
	UpdateUser(sel bson.M, user *User) error
}

type MongoUserStore struct {
	address   string
	session   *mgo.Session
	usersData *mgo.Collection
}

func (mongo *MongoUserStore) UpdateUser(sel bson.M, user *User) error {
	return mongo.usersData.Update(sel, user)
}

func (mongo *MongoUserStore) GetUserByDevice(device bson.M, user *User) {
	mongo.usersData.Find(device).One(user)
	return
}

func (mongo *MongoUserStore) Update(sel, update bson.M) error {
	return mongo.usersData.Update(sel, update)
}

func (mongo *MongoUserStore) FindUser(query bson.M, user *User) error {
	return mongo.usersData.Find(query).One(user)
}

func (mongo *MongoUserStore) Insert(user User) error {
	return mongo.usersData.Insert(user)
}

func InitUserStore(address string) (UserStore, error) {
	uStore := &MongoUserStore{
		address: address,
	}
	session, err := mgo.Dial(address)
	if err != nil {
		return nil, err
	}
	uStore.session = session
	uStore.usersData = uStore.session.DB(dbUsers).C(tableUsers)
	return uStore, nil
}

func (mongoUserData *MongoUserStore) Close() error {
	mongoUserData.session.Close()
	return nil
}
