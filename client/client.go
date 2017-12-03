package client

import (
	"log"
	"sync"

	socketio "github.com/googollee/go-socket.io"

	"github.com/Appscrunch/Multy-back/btc"
)

type ConnectedPool struct {
	clients map[string]*Client // socketio connections by client id
	m       *sync.RWMutex
}

func InitConnectedPool() *ConnectedPool {
	return &ConnectedPool{
		m:       &sync.RWMutex{},
		clients: make(map[string]*Client, 0),
	}
}

func (cp *ConnectedPool) AddClient(clientID string, clientObj *Client) {
	cp.m.Lock()
	defer cp.m.Unlock()

	(cp.clients[clientID]) = clientObj
}

func (cp *ConnectedPool) RemoveClient(clientID string) {
	cp.m.Lock()
	defer cp.m.Unlock()

	delete(cp.clients, clientID)
}

// Client is a struct with client data
type Client struct {
	clientID int64
	token    string
	wsConns  map[string]socketio.Conn //connections by connection ID

	btcCh chan btc.BtcTransaction
}

func newClient(id string, rawData []byte, connCh chan btc.BtcTransaction, conn socketio.Conn) *Client {
	// TODO: add marshaling data from connection
	// client id, device id, etc
	newClient := &Client{
		clientID: 1234,
		wsConns:  make(map[string]socketio.Conn, 0),
	}
	newClient.wsConns[id] = conn
	return newClient
}

func (c *Client) listenBTC() {
	var newTransaction btc.BtcTransaction

	for {
		select {
		case newTransaction = <-c.btcCh:
			log.Printf("got new transaction: %+v\n", newTransaction)
			for _, conn := range c.wsConns {
				conn.Emit("newTransaction", newTransaction)
			}
		}
	}
}
