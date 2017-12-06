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
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		userInfo, err := getHeaderDataSocketIO(s.RemoteHeader())
		if err != nil {
			log.Printf("[ERR] get socketio headers: %s\n", err.Error())
			return fmt.Errorf("get socketio headers: %s", err.Error())
		}

		connectionID := s.ID()
		userID := userInfo.userID

		newConn := newSocketIOUser(connectionID, userInfo, btcCh, s)
		users.AddUserConn(userID, newConn)

		return nil
	})

	server.OnError("/", func(err error) {
		fmt.Println("[ERR] socketio: ", err)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)

	go func() {
		http.ListenAndServe("0.0.0.0:7779", nil)
	}()
	return nil, nil
}
