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
	userID := headers.Get("userID")
	if len(userID) == 0 {
		return nil, fmt.Errorf("wrong userID header")
	}

	deviceType := headers.Get("deviceType")
	if len(deviceType) == 0 {
		return nil, fmt.Errorf("wrong deviceType header")
	}

	jwtToken := headers.Get("jwtToken")
	if len(jwtToken) == 0 {
		return nil, fmt.Errorf("wrong jwtToken header")
	}

	return &SocketIOUser{
		userID:     userID,
		deviceType: deviceType,
		jwtToken:   jwtToken,
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
		log.Panic(http.ListenAndServe("0.0.0.0:7778", serveMux))
	}()
	return nil, nil
}
