package client

import (
	"fmt"
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

func SetRestHandlers(userDB store.UserStore, r *gin.Engine, clientRPC *rpcclient.Client) (*RestClient, error) {

	restClient := &RestClient{
		userStore: userDB,
		rpcClient: clientRPC,
		m:         &sync.Mutex{},
	}

	initMiddlewareJWT(restClient)

	r.POST("/auth", restClient.LoginHandler())

	v1 := r.Group("/api/v1")
	v1.Use(restClient.middlewareJWT.MiddlewareFunc())
	{
		v1.POST("/wallet", restClient.addWallet())

		v1.POST("/address", restClient.addAddress())

		v1.GET("/wallets", restClient.getWallets())

		v1.GET("/transaction/feerate", restClient.getFeeRate())

		v1.GET("/outputs/spendable/:walletIndex", restClient.getSpendableOutputs())

		v1.GET("/getexchangeprice/:from/:to", restClient.getExchangePrice()) // rgb(255, 0, 0) полностью меняем

		v1.POST("/transaction/send", restClient.sendRawTransaction())
		// v1.GET("/outputs/spendable", restClient.getSpendbleOutputs())
		//

		//
		// v1.POST("/exchange/:exchanger", restClient.exchange())
		// v1.POST("/sendtransaction/:curid", restClient.sendTransaction())
		//
		// // r.GET("/getassetsinfo", restClient.getAssetsInfo()) err rgb(255, 0, 0)
		//
		// v1.GET("/getblock/:height", restClient.getBlock())
		// // r.GET("/getmarkets", restClient.getMarkets())
		// v1.GET("/gettransactioninfo/:txid", restClient.getTransactionInfo())
		// v1.GET("/gettickets/:pair", restClient.getTickets())
		//

		//
		// v1.GET("/getexchangeprice/:from/:to", restClient.getExchangePrice())
		// // r.GET("/getactivities/:adress/:datefrom/:dateto", restClient.getActivities())
		// v1.GET("/getaddressbalance/:addr", restClient.getAdressBalance())
	}
	return restClient, nil
}

func initMiddlewareJWT(restClient *RestClient) {
	restClient.middlewareJWT = &GinJWTMiddleware{
		Realm:      "test zone",
		Key:        []byte("secret key"), // config
		Timeout:    time.Hour,
		MaxRefresh: time.Hour,
		Authenticator: func(userId, deviceId, pushToken string, deviceType int, c *gin.Context) (store.User, bool) {
			query := bson.M{"userID": userId}

			user := store.User{}

			err := restClient.userStore.FindUser(query, &user)

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
	CurrencyID   int    `json:"currencyID"`
	Address      string `json:"address"`
	AddressIndex int    `json:"addressIndex"`
	WalletIndex  int    `json:"walletIndex"`
	WalletName   string `json:"walletName"`
}

// ----------addAddress------------
// ----------deleteWallet----------
type SelectWallet struct {
	WalletIndex  int    `json:"walletIndex"`
	Address      string `json:"address"`
	AddressIndex int    `json:"addressIndex"`
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

// ----------getAllUserAssets----------
type WalletExtended struct {
	CuurencyID  int         `bson:"chain"`       //cuurencyID
	WalletIndex int         `bson:"walletIndex"` //walletIndex
	Addresses   []AddressEx `bson:"adresses"`
}

type AddressEx struct { // extended
	AddressID int    `bson:"addressID"` //addressIndex
	Address   string `bson:"address"`
	Amount    int    `bson:"amount"` //float64
}

func (restClient *RestClient) addWallet() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]

		var wp WalletParams
		decodeBody(c, &wp)

		wallet := createWallet(wp.CurrencyID, wp.Address, wp.AddressIndex, wp.WalletIndex, wp.WalletName)

		sel := bson.M{"devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets": wallet}}

		restClient.m.Lock()
		err := restClient.userStore.Update(sel, update)
		restClient.m.Unlock()
		if err != nil {
			c.JSON(http.StatusCreated, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"code":    http.StatusCreated,
			"message": http.StatusText(http.StatusCreated),
		})
	}
}

func (restClient *RestClient) addAddress() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]

		var sw SelectWallet
		decodeBody(c, &sw)

		addr := store.Address{
			Address:      sw.Address,
			AddressIndex: sw.AddressIndex,
		}

		sel := bson.M{"wallets.walletIndex": sw.WalletIndex, "devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets.$.adresses": addr}}

		restClient.m.Lock()
		err := restClient.userStore.Update(sel, update)
		restClient.m.Unlock()
		if err != nil {
			c.JSON(http.StatusCreated, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": http.StatusText(http.StatusOK),
		})
	}
}

func (restClient *RestClient) getWallets() gin.HandlerFunc { //recieve rgb(255, 0, 0)
	return func(c *gin.Context) {

		token := strings.Split(c.GetHeader("Authorization"), " ")[1]
		sel := bson.M{"devices.JWT": token}

		var user store.User

		err := restClient.userStore.FindUser(sel, &user)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusNotFound,
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"userWallets": user.Wallets,
			})
		}

	}
}
func (restClient *RestClient) getFeeRate() gin.HandlerFunc {
	return func(c *gin.Context) {

		var rates []store.RatesRecord
		speeds := []int{
			1, 2, 3, 4, 5,
		}
		err := restClient.userStore.GetAllRates("category", &rates)

		fmt.Println(rates)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"VerySlow": rates[speeds[0]].Category,
				"Slow":     rates[speeds[1]].Category,
				"Medium":   rates[speeds[2]].Category,
				"Fast":     rates[speeds[3]].Category,
				"VeryFast": rates[speeds[4]].Category,
			})
		}

	}
}

func (restClient *RestClient) getSpendableOutputs() gin.HandlerFunc {
	return func(c *gin.Context) {
		walletIndex, _ := strconv.Atoi(c.Param("walletIndex"))
		token := strings.Split(c.GetHeader("Authorization"), " ")[1]

		var user store.User

		sel := bson.M{"devices.JWT": token, "wallets.walletIndex": walletIndex}

		err := restClient.userStore.FindUser(sel, &user)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}

		outWallet := store.Wallet{}

	loop:
		for _, wallet := range user.Wallets {
			if wallet.WalletIndex == walletIndex {
				outWallet = wallet
				break loop
			}
		}

		if len(outWallet.Adresses) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusBadRequest,
				"message": "No spendable outputs",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"outs": outWallet,
			})
		}

	}
}

func (restClient *RestClient) sendRawTransaction() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawTx := RawTx{}
		decodeBody(c, &rawTx)
		txid, err := restClient.rpcClient.SendCyberRawTransaction(rawTx.Transaction, rawTx.AllowHighFees)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"TransactionHash": txid,
			})
		}
	}
}

type RawTx struct { // remane RawClientTransaction
	Transaction   string `json:"transaction"`   //HexTransaction
	AllowHighFees bool   `json:"allowHighFees"` // hardcode true, remove field
}

func (restClient *RestClient) blank() gin.HandlerFunc {
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
