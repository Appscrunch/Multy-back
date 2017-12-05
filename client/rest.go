package client

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Appscrunch/Multy-back/currencies"
	"github.com/Appscrunch/Multy-back/store"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type RestClient struct {
	middlewareJWT *GinJWTMiddleware
	userStore     store.UserStore
	rpcClient     *rpcclient.Client
	m             *sync.Mutex
}

func SetRestHandlers(userDB store.UserStore, r *gin.RouterGroup, clientRPC *rpcclient.Client) (*RestClient, error) {
	restClient := &RestClient{
		userStore: userDB,
		rpcClient: clientRPC,
		m:         &sync.Mutex{},
	}

	initMiddlewareJWT(restClient)

	v1 := r.Group("/api/v1")
	v1.Use(restClient.middlewareJWT.MiddlewareFunc())
	{
		v1.POST("/auth", restClient.LoginHandler())

		v1.POST("/addadress/:userid", restClient.addAddress())
		v1.POST("/addwallet/:userid", restClient.addWallet()) // rename
		v1.POST("/exchange/:exchanger", restClient.exchange())
		v1.POST("/sendtransaction/:curid", restClient.sendTransaction())

		// r.GET("/getassetsinfo", restClient.getAssetsInfo()) err rgb(255, 0, 0)

		v1.GET("/getblock/:height", restClient.getBlock())
		// r.GET("/getmarkets", restClient.getMarkets())
		v1.GET("/gettransactioninfo/:txid", restClient.getTransactionInfo())
		v1.GET("/gettickets/:pair", restClient.getTickets())

		v1.GET("/getadresses/:id", restClient.getAddresses())

		v1.GET("/getexchangeprice/:from/:to", restClient.getExchangePrice())
		// r.GET("/getactivities/:adress/:datefrom/:dateto", restClient.getActivities())
		v1.GET("/getaddressbalance/:addr", restClient.getAdressBalance())
	}
	return restClient, nil
}

func initMiddlewareJWT(restClient *RestClient) {
	restClient.middlewareJWT = &GinJWTMiddleware{
		Realm:      "test zone",
		Key:        []byte("secret key"), // config
		Timeout:    time.Hour,
		MaxRefresh: time.Hour,
		Authenticator: func(userId string, deviceId string, password string, c *gin.Context) (store.User, bool) {

			query := bson.M{"userID": userId}

			user := store.User{}

			err := restClient.userStore.Find(query, &user)

			if err != nil || len(user.UserID) == 0 {
				return store.User{}, false
			}

			return user, true
		},
		//Authorizator:  authorizator,
		//Unauthorized: unauthorized,
		Unauthorized: nil,
		TokenLookup:  "header:Authorization",

		TokenHeadName: "Bearer",

		TimeFunc: time.Now,
	}
}

// ----------createWallet----------
type WalletParams struct {
	Currency  int    `json:"currency"`
	Address   string `json:"address"`
	AddressID string `json:"addressID"`
	WalletID  string `json:"walletID"`
}

// ----------addAddress------------
// ----------deleteWallet----------
type selectWallet struct {
	WalletID  string `json:"walletID"`
	Address   string `json:"address"`
	AddressID string `json:"addressID"`
}
type Address struct {
	Address   string `json:"address"`
	AddressID string `json:"addressID"`
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

func (restClient *RestClient) getBlock() gin.HandlerFunc {
	return func(c *gin.Context) {
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
}

/*
var getMarkets = func(c *gin.Context) {

}
*/

func (restClient *RestClient) getTickets() gin.HandlerFunc { // shapeshift
	return func(c *gin.Context) {
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
}

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

func (restClient *RestClient) getAddresses() gin.HandlerFunc { //recieve rgb(255, 0, 0)
	return func(c *gin.Context) {

		token := strings.Split(c.GetHeader("Authorization"), " ")[1]
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

		c.JSON(http.StatusOK, gin.H{
			"userWallets": user.Wallets,
		})
	}
}

func (restClient *RestClient) getExchangePrice() gin.HandlerFunc {
	return func(c *gin.Context) {
		from := strings.ToUpper(c.Param("from"))
		to := strings.ToUpper(c.Param("to"))

		url := "https://min-api.cryptocompare.com/data/price?fsym=" + from + "&tsyms=" + to
		var er map[string]interface{}
		makeRequest(c, url, &er)

		c.JSON(http.StatusOK, gin.H{
			to: er[to],
		})
	}
}

// var getAssetsInfo = func(c *gin.Context) { // recieve rgb(255, 0, 0)
// 	token := strings.Split(c.GetHeader("Authorization"), " ")[1]
// 	dv := getaddresses(c, token)
// 	assets := map[int]int{} // name of wallet to total ballance
// 	excPrice := map[string]float64{}
// 	for name, as := range dv {
// 		switch as.Chain {
// 		case currencies.String(currencies.Bitcoin):
// 			for _, addr := range as.Adresses {
// 				assets[name] += getadressbalance(c, addr.Address, 1)
// 			}
// 		case currencies.String(currencies.Ether):
// 			for _, addr := range as.Adresses {
// 				assets[name] += getadressbalance(c, addr.Address, 2)
// 			}
// 		default:
// 			fmt.Println(strings.ToUpper(as.Chain))
// 		}
// 	}
// 	excPrice["BTC"] = getexchangeprice(c, "BTC", "USD")
// 	excPrice["ETH"] = getexchangeprice(c, "ETH", "USD")
//
// 	c.JSON(http.StatusOK, gin.H{
// 		"wallets":       dv,
// 		"ballances":     assets,
// 		"exchangePrice": excPrice,
// 	})
// }

// var getActivities = func(c *gin.Context) { // следующий спринт
//
// }
func (restClient *RestClient) getTransactionInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
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

func (restClient *RestClient) addWallet() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]

		var wp WalletParams
		decodeBody(c, &wp)
		wallet := createWallet(currencies.String(wp.Currency), wp.Address, wp.AddressID, wp.WalletID)

		sel := bson.M{"devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets": wallet}}

		restClient.m.Lock()
		err := restClient.userStore.Update(sel, update)
		restClient.m.Unlock()
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

		var sw selectWallet
		decodeBody(c, &sw)

		addr := Address{
			Address:   sw.Address,
			AddressID: sw.AddressID,
		}

		sel := bson.M{"wallets.walletid": sw.WalletID, "devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets.$.adresses": addr}}

		restClient.m.Lock()
		err := restClient.userStore.Update(sel, update)
		restClient.m.Unlock()

		responseErr(c, err, http.StatusInternalServerError) // 500

		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": http.StatusText(http.StatusOK),
		})
	}
}

func (restClient *RestClient) sendTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		curID := c.Param("curid")
		i, err := strconv.Atoi(curID)
		if err != nil {
			i = 1337 // err
		}

		switch i {
		case currencies.Bitcoin:
			var tx Tx
			decodeBody(c, &tx)
			txid, err := restClient.rpcClient.SendCyberRawTransaction(tx.Transaction, tx.AllowHighFees)
			responseErr(c, err, http.StatusInternalServerError) // 500

			c.JSON(http.StatusOK, gin.H{
				"txid": txid,
			})

		case currencies.Ether:
			//not implemented yet
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": http.StatusText(http.StatusBadRequest),
			})
		}
	}
}
