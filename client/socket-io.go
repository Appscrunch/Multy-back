package client

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

const (
	socketIOOutMsg = "outcoming"
	socketIOInMsg  = "incoming"

	deviceTypeMac     = "mac"
	deviceTypeAndroid = "android"
)

type SocketIONotifyMessage struct {
	TransactionType string `json:"transactionType"`
	Amount          int64  `json:"amount"`
	TxID            int64  `json:"txid"`
}

type SocketIOIdentifyer struct {
	userID     string
	deviceType string
	jwtToken   []byte
}

/*func getHeaderDataSocketIO(headers http.Header) (*SocketIOIdentifyer, error) {
	if _, ok := headers[""]
	return nil, nil
}*/

func SetSocketIOHandlers(r *gin.RouterGroup, clients *ConnectedPool) (*socketio.Server, error) {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		//userIdentificator, err := getHeaderDataSocketIO(s.RemoteHeader())
		/*	for _, h := range headers {
			log.Printf("DEBUG header %+v\n", h)
		}*/
		connectionID := s.ID()
		connCh := make(chan btc.BtcTransaction, 0)

		// TODO: from connection header
		clientID := "clientID"

		newConn := newClient(connectionID, []byte("data"), connCh, s)
		clients.AddClient(connectionID, newConn)
		go newConn.listenBTC()

		clients.AddClient(clientID, newConn)

		return nil
	})

	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)

	go func() {
		http.ListenAndServe("0.0.0.0:7778", nil)
	}()
	return nil, nil
}
