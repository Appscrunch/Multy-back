package client

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Appscrunch/Multy-back/btc"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/graarh/golang-socketio/transport"

	"github.com/graarh/golang-socketio"
)

const (
	socketIOOutMsg = "outcoming"
	socketIOInMsg  = "incoming"

	deviceTypeMac     = "mac"
	deviceTypeAndroid = "android"
)

func getHeaderDataSocketIO(headers http.Header) (*SocketIOUser, error) {
	if _, ok := headers["userID"]; !ok {
		return nil, fmt.Errorf("wrong userID header")
	}
	userID := headers["userID"]
	if len(userID[0]) == 0 {
		return nil, fmt.Errorf("wrong userID header")
	}

	if _, ok := headers["deviceType"]; !ok {
		return nil, fmt.Errorf("wrong deviceType header")
	}
	deviceType := headers["deviceType"]
	if len(userID[0]) == 0 {
		return nil, fmt.Errorf("wrong deviceType header")
	}

	if _, ok := headers["jwtToken"]; !ok {
		return nil, fmt.Errorf("wrong jwtToken header")
	}
	jwtToken := headers["jwtToken"]
	if len(userID[0]) == 0 {
		return nil, fmt.Errorf("wrong jwtToken header")
	}

	return &SocketIOUser{
		userID:     userID[0],
		deviceType: deviceType[0],
		jwtToken:   jwtToken[0],
	}, nil
}

func SetSocketIOHandlers(r *gin.RouterGroup, btcCh chan btc.BtcTransactionWithUserID, users *SocketIOConnectedPool) (*socketio.Server, error) {
	server := gosocketio.NewServer(transport.GetDefaultWebsocketTransport())

	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		fmt.Println("connected:", c.Id())
		userInfo, err := getHeaderDataSocketIO(c.RequestHeader())
		if err != nil {
			log.Printf("[ERR] get socketio headers: %s\n", err.Error())
			return
		}

		connectionID := c.Id()
		userID := userInfo.userID

		newConn := newSocketIOUser(connectionID, userInfo, btcCh, c)
		users.AddUserConn(userID, newConn)

		return
	})

	server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		log.Println("Disconnected")
	})

	serveMux := http.NewServeMux()
	serveMux.Handle("/socket.io/", server)

	log.Println("Starting server...")
	go func() {
		log.Panic(http.ListenAndServe(":7778", serveMux))
	}()
	return nil, nil
}
