package client

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/Appscrunch/Multy-back/store"
	"github.com/Appscrunch/Multy-back/types"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type RestClient struct {
	userStore store.UserStore
	rpcClient *rpcclient.Client
	m         *sync.Mutex
}

func SetRestHandlers(userDB store.UserStore, r *gin.RouterGroup, clientRPC *rpcclient.Client) (*RestClient, error) {
	restClient := &RestClient{
		userStore: userDB,
		rpcClient: clientRPC,
		m:         &sync.Mutex{},
	}

	//r.POST("/auth", authMiddleware.LoginHandler)
	//r.Use(authMiddleware.MiddlewareFunc())

	r.POST("/addadress/:userid", restClient.addAddress())
	r.POST("/addwallet/:userid", restClient.addWallet()) // rename
	r.POST("/exchange/:exchanger", restClient.exchange())
	//r.POST("/sendtransaction/:curid", sendTransaction)

	/*r.GET("/getassetsinfo", getAssetsInfo)
	r.GET("/getblock/:height", getBlock)
	r.GET("/getmarkets", getMarkets)
	r.GET("/gettransactioninfo/:txid", getTransactionInfo)
	r.GET("/gettickets/:pair", getTickets)
	r.GET("/getadresses/:id", getAdresses)
	r.GET("/getexchangeprice/:from/:to", getExchangePrice)
	r.GET("/getactivities/:adress/:datefrom/:dateto", getActivities)*/
	r.GET("/getaddressbalance/:addr", restClient.getAdressBalance())

	return restClient, nil
}

// ----------createWallet----------
type WalletParams struct {
	Currency  int    `json:"currency"`
	Address   string `json:"address"`
	AddressID string `json:"addressID"`
}

// ----------addAddress------------
// ----------deleteWallet----------
type selectWallet struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// ----------sendTransaction----------
type Tx struct {
	Transaction   string `json:"transaction"`
	AllowHighFees bool   `json:"allowHighFees"`
}

// ----------getAdresses----------
type DisplayWallet struct {
	Chain    string          `json:"chain"`
	Adresses []store.Address `json:"adresses"`
}

func (restClient *RestClient) exchange() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "not implemented",
		})
	}
}

/*
var getBlock = func(c *gin.Context) {
	height := c.Param("height")

	url := "https://bitaps.com/api/block/" + height

	if height == "last" {
		url = "https://bitaps.com/api/block/latest"
	}

	response, err := http.Get(url)
	responseErr(c, err, http.StatusServiceUnavailable) // 503

	data, err := ioutil.ReadAll(response.Body)
	responseErr(c, err, http.StatusInternalServerError) // 500

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Write(data)
}

var getMarkets = func(c *gin.Context) {

}

var getTickets = func(c *gin.Context) { // shapeshift
	pair := c.Param("pair")
	url := "https://shapeshift.io/marketinfo/" + pair

	var to map[string]interface{}

	makeRequest(c, url, &to)
	// fv := v.Convert(floatType).Float()

	c.JSON(http.StatusOK, gin.H{
		"pair":     to["pair"],
		"rate":     to["rate"],
		"limit":    to["limit"],
		"minimum":  to["minimum"],
		"minerFee": to["minerFee"],
	})

}
*/

func (restClient *RestClient) getAdressBalance() gin.HandlerFunc {
	return func(c *gin.Context) { //recieve rgb(255, 0, 0)
		addr := c.Param("addr")
		url := "https://blockchain.info/q/addressbalance/" + addr
		response, err := http.Get(url)
		responseErr(c, err, http.StatusServiceUnavailable) // 503

		data, err := ioutil.ReadAll(response.Body)
		responseErr(c, err, http.StatusInternalServerError) // 500

		b, err := strconv.Atoi(string(data))

		c.JSON(http.StatusOK, gin.H{
			"ballance": b,
		})
	}
}

func (restClient *RestClient) getAddresses() gin.HandlerFunc {
	return func(c *gin.Context) { //recieve rgb(255, 0, 0)
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]
		// userID := c.Param("userid")
		sel := bson.M{"devices.JWT": token}

		var user store.User

		restClient.userStore.GetUserByDevice(sel, user)

		if user.UserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": http.StatusText(400),
			})
			return
		}

		userWallets := map[int]DisplayWallet{}

		for k, v := range user.Wallets {
			userWallets[k] = DisplayWallet{v.Chain, v.Adresses}
		}

		c.JSON(http.StatusOK, gin.H{
			"userWallets": userWallets,
		})
	}
}

/*
var getExchangePrice = func(c *gin.Context) { //recieve rgb(255, 0, 0)

	from := strings.ToUpper(c.Param("from"))
	to := strings.ToUpper(c.Param("to"))

	url := "https://min-api.cryptocompare.com/data/price?fsym=" + from + "&tsyms=" + to
	var er map[string]interface{}
	makeRequest(c, url, &er)

	c.JSON(http.StatusOK, gin.H{
		to: er[to],
	})

}

var getAssetsInfo = func(c *gin.Context) { // recieve rgb(0, 255, 0)
	token := strings.Split(c.GetHeader("Authorization"), " ")[1]
	dv := getaddresses(c, token)
	assets := map[int]int{} // name of wallet to total ballance
	excPrice := map[string]float64{}
	for name, as := range dv {
		switch as.Chain {
		case types.String(types.Bitcoin):
			for _, addr := range as.Adresses {
				assets[name] += getadressbalance(c, addr.Address, 1)
			}
		case types.String(types.Ether):
			for _, addr := range as.Adresses {
				assets[name] += getadressbalance(c, addr.Address, 2)
			}
		default:
			fmt.Println(strings.ToUpper(as.Chain))
		}
	}
	excPrice["BTC"] = getexchangeprice(c, "BTC", "USD")
	excPrice["ETH"] = getexchangeprice(c, "ETH", "USD")

	c.JSON(http.StatusOK, gin.H{
		"wallets":       dv,
		"ballances":     assets,
		"exchangePrice": excPrice,
	})
}

var getActivities = func(c *gin.Context) { // следующий спринт

}

var getTransactionInfo = func(c *gin.Context) {
	txid := c.Param("txid")
	url := "https://bitaps.com/api/transaction/" + txid
	response, err := http.Get(url)
	responseErr(c, err, http.StatusServiceUnavailable) // 503

	data, err := ioutil.ReadAll(response.Body)
	responseErr(c, err, http.StatusInternalServerError) // 500

	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Write(data)
}

// var deleteWallet = func(c *gin.Context) {
// 	token := strings.Split(c.GetHeader("Authorization"), " ")[1]
// 	userID := c.Param("userid")
//
// 	var sw selectWallet
// 	decodeBody(c, &sw)
//
// 	sel := bson.M{"wallets.name": sw.Name, "userID": userID, "devices.JWT": token}
// 	update := bson.M{"$push": bson.M{"wallets": wallet}}
//
// }
*/

func (restClient *RestClient) addWallet() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]
		userID := c.Param("userid")

		var wp WalletParams
		decodeBody(c, &wp)
		wallet := createWallet(types.String(wp.Currency), wp.Address, wp.AddressID)

		sel := bson.M{"devices.JWT": token, "userID": userID}
		update := bson.M{"$push": bson.M{"wallets": wallet}}

		restClient.m.Lock()
		defer restClient.m.Unlock()
		err := restClient.userStore.Update(sel, update)
		responseErr(c, err, http.StatusInternalServerError) // 500

		c.JSON(http.StatusCreated, gin.H{
			"code":    http.StatusCreated,
			"message": http.StatusText(http.StatusCreated),
		})
	}
}

func (restClient *RestClient) addAddress() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]
		userID := c.Param("userid")

		var sw selectWallet
		decodeBody(c, &sw)

		sel := bson.M{"wallets.name": sw.Name, "userID": userID, "devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets.$.adresses": sw.Address}}

		restClient.m.Lock()
		defer restClient.m.Unlock()
		err := restClient.userStore.Update(sel, update)
		responseErr(c, err, http.StatusInternalServerError) // 500

		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": http.StatusText(http.StatusOK),
		})
	}
}

/*
var sendTransaction = func(c *gin.Context) {
	curID := c.Param("curid")
	i, err := strconv.Atoi(curID)
	if err != nil {
		i = 1337 // err
	}

	switch i {
	case types.Bitcoin:
		var tx Tx
		decodeBody(c, &tx)
		txid, err := restClient.SendCyberRawTransaction(tx.Transaction, tx.AllowHighFees)
		responseErr(c, err, http.StatusInternalServerError) // 500

		c.JSON(http.StatusOK, gin.H{
			"txid": txid,
		})

	case types.Ether:
		//not implemented yet
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": http.StatusText(http.StatusBadRequest),
		})
	}

}*/
