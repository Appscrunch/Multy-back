package client

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

func SetSocketIOHandlers(r *gin.RouterGroup, clients *ConnectedPool) (*socketio.Server, error) {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())

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

	log.Fatal(http.ListenAndServe(":7778", nil))

	return nil, nil
}
