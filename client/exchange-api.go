package client

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
)

// GdaxAPI wraps websocket connection for GdaxAPI.
type GdaxAPI struct {
	conn  *websocket.Conn
	rates *Rates
}

//GDAXSocketEvent is a GDAX json parser structure
type GDAXSocketEvent struct {
	ProductID string `json:"product_id"`
	Price     string `json:"price"`
}

func (eChart *exchangeChart) initGdaxAPI() (*GdaxAPI, error) {
	u := url.URL{Scheme: "wss", Host: "ws-feed.gdax.com", Path: ""}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Println("dial:", err)
		return nil, err
	}

	// TODO: Add Close() method
	// defer c.Close()
	// done := make(chan struct{})
	// defer ticker.Stop()

	subscribtion := `{"type":"subscribe","channels":[{"name":"ticker_1000","product_ids":["BTC-USD","BTC-EUR","ETH-BTC","ETH-USD","ETH-EUR"]}]}`
	c.WriteMessage(websocket.TextMessage, []byte(subscribtion))

	// defer c.Close()
	// defer close(done)
	return &GdaxAPI{c, eChart.rates}, nil
}

func (gdax *GdaxAPI) listen() {
	for {
		_, message, err := gdax.conn.ReadMessage()
		if err != nil {
			log.Println("read message:", err)
			continue
		}

		rateRaw := &GDAXSocketEvent{}
		err = json.Unmarshal(message, rateRaw)
		if err != nil {
			log.Println("Unmarshal marshal ")
			continue
		}
		gdax.updateRate(rateRaw)
	}
}

func (gdax *GdaxAPI) updateRate(rawRate *GDAXSocketEvent) {
	floatPrice, err := strconv.ParseFloat(rawRate.Price, 32)
	if err != nil {
		log.Printf("ParseFloat %s: %s\n", rawRate.Price, err.Error())
		return
	}

	gdax.rates.m.Lock()

	switch rawRate.ProductID {
	case "BTC-USD":
		gdax.rates.exchangeSingle.BTCtoUSD = floatPrice
		gdax.rates.exchangeSingle.USDtoBTC = 1 / floatPrice
	case "BTC-EUR":
		gdax.rates.exchangeSingle.EURtoBTC = 1 / floatPrice
	case "ETH-BTC":
		gdax.rates.exchangeSingle.ETHtoBTC = floatPrice
		//gdax.rates.exchangeSingle.BTCtoETH = 1 / floatPrice
	case "ETH-USD":
		gdax.rates.exchangeSingle.ETHtoUSD = floatPrice
	case "ETH-EUR":
		gdax.rates.exchangeSingle.ETHtoEUR = floatPrice
	default:
		log.Printf("unknown rate: %+v\n", rawRate)
	}

	gdax.rates.m.Unlock()
}
