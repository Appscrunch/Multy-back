package client

import (
	"log"
	"sync"

	socketio "github.com/googollee/go-socket.io"

	"github.com/Appscrunch/Multy-back/btc"
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
	var newTransaction btc.BtcTransactionWithUserID

	for {
		select {
		case newTransactionWithUserID := <-sConnPool.btcCh:
			log.Printf("got new transaction: %+v\n", newTransaction)
			if _, ok := sConnPool.users[newTransactionWithUserID.UserID]; !ok {
				break
			}
			userID := newTransactionWithUserID.UserID
			userConns := sConnPool.users[userID].conns
			for _, conn := range userConns {
				conn.Emit("newTransaction", newTransaction.NotificationMsg)
			}
		}
	}
}

func (sConnPool *SocketIOConnectedPool) AddUserConn(userID string, userObj *SocketIOUser) {
	sConnPool.m.Lock()
	defer sConnPool.m.Unlock()

	(sConnPool.users[userID]) = userObj
}

func (sConnPool *SocketIOConnectedPool) RemoveUserConn(userID string) {
	sConnPool.m.Lock()
	defer sConnPool.m.Unlock()

	delete(sConnPool.users, userID)
}

type SocketIOUser struct {
	userID     string
	deviceType string
	jwtToken   string

	conns map[string]socketio.Conn
}

func newSocketIOUser(id string, connectedUser *SocketIOUser, btcCh chan btc.BtcTransactionWithUserID, conn socketio.Conn) *SocketIOUser {
	connectedUser.conns = make(map[string]socketio.Conn, 0)
	connectedUser.conns[id] = conn

	return connectedUser
}
