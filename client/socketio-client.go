package client

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/graarh/golang-socketio"
)

type SocketIOConnectedPool struct {
	users map[string]*SocketIOUser // socketio connections by client id
	m     *sync.RWMutex

	btcCh chan btc.BtcTransactionWithUserID
}

func InitConnectedPool(btcCh chan btc.BtcTransactionWithUserID) *SocketIOConnectedPool {
	pool := &SocketIOConnectedPool{
		m:     &sync.RWMutex{},
		users: make(map[string]*SocketIOUser, 0),
		btcCh: btcCh,
	}
	go pool.listenBTC()
	return pool
}

func (sConnPool *SocketIOConnectedPool) listenBTC() {
	var newTransactionWithUserID btc.BtcTransactionWithUserID

	for {
		select {
		case newTransactionWithUserID = <-sConnPool.btcCh:
			log.Printf("got new transaction: %+v\n", newTransactionWithUserID)
			if _, ok := sConnPool.users[newTransactionWithUserID.UserID]; !ok {
				break
			}
			userID := newTransactionWithUserID.UserID
			//TODO: with mutex
			userConns := sConnPool.users[userID].conns
			log.Printf("userConn=%+v\n", userConns)

			/*	var cc *SocketIOUser
				for _, c := range sConnPool.users {
					cc = c
					break
				}
				if cc == nil {
					break
				}*/
			for _, conn := range userConns {
				//for _, conn := range userConns {
				log.Println("id=", conn.Id())
				msgRaw, err := json.Marshal(newTransactionWithUserID.NotificationMsg)
				if err != nil {
					break
				}
				conn.Emit("/newTransaction", string(msgRaw))
			}
		}
	}
}

func (sConnPool *SocketIOConnectedPool) AddUserConn(userID string, userObj *SocketIOUser) {
	log.Println("DEBUG AddUserConn: ", userID)
	sConnPool.m.Lock()
	defer sConnPool.m.Unlock()

	(sConnPool.users[userID]) = userObj
}

func (sConnPool *SocketIOConnectedPool) RemoveUserConn(userID string) {
	log.Println("DEBUG RemoveUserConn: ", userID)
	sConnPool.m.Lock()
	defer sConnPool.m.Unlock()

	delete(sConnPool.users, userID)
}

type SocketIOUser struct {
	userID     string
	deviceType string
	jwtToken   string

	conns map[string]*gosocketio.Channel
}

func newSocketIOUser(id string, connectedUser *SocketIOUser, btcCh chan btc.BtcTransactionWithUserID, conn *gosocketio.Channel) *SocketIOUser {
	connectedUser.conns = make(map[string]*gosocketio.Channel, 0)
	connectedUser.conns[id] = conn

	return connectedUser
}

func (user *SocketIOUser) Subscribe() {
	log.Println("[DEBUG] Subscribe: not implemented")
	/*for {
		config := nsq.NewConfig()
		q, err := nsq.NewConsumer("socketIO", "getExchange", config)
		if err != nil {
			log.Printf("[ERR] Subscribe: %s/tuserID=%s\n", err.Error(), user.userID)
			return
		}

		q.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
			log.Printf("Got a message: %v", message)

			con
		}))
		/*err := q.ConnectToNSQD("127.0.0.1:4150")
		if err != nil {
			log.Panic("Could not connect")
		}
	}*/
}
