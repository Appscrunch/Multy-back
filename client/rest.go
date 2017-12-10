package client

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Appscrunch/Multy-back/currencies"
	"github.com/Appscrunch/Multy-back/store"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

const (
	msgErrHeaderError          = "wrong authorization headers"
	msgErrRequestBodyError     = "missing request body params"
	msgErrUserNotFound         = "user not found in db"
	msgErrRatesError           = "internal server error rates"
	msgErrDecodeWalletIndexErr = "wrong wallet index"
	msgErrNoSpendableOuts      = "No spendable outputs"
)

type RestClient struct {
	middlewareJWT *GinJWTMiddleware
	userStore     store.UserStore
	rpcClient     *rpcclient.Client
}

func SetRestHandlers(userDB store.UserStore, r *gin.Engine, clientRPC *rpcclient.Client) (*RestClient, error) {

	restClient := &RestClient{
		userStore: userDB,
		rpcClient: clientRPC,
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
		// // r.GET("/getactivities/:address/:datefrom/:dateto", restClient.getActivities())
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
type SelectWallet struct {
	WalletIndex  int    `json:"walletIndex"`
	Address      string `json:"address"`
	AddressIndex int    `json:"addressIndex"`
}

// ----------getFeeRate----------
type EstimationSpeeds struct {
	VerySlow int
	Slow     int
	Medium   int
	Fast     int
	VeryFast int
}

// ----------sendTransaction----------
type Tx struct {
	Transaction   string `json:"transaction"`
	AllowHighFees bool   `json:"allowHighFees"`
}

// ----------getAdresses----------
type DisplayWallet struct {
	Chain    string          `json:"chain"`
	Adresses []store.Address `json:"addresses"`
}

// ----------getAllUserAssets----------
type WalletExtended struct {
	CuurencyID  int         `bson:"chain"`       //cuurencyID
	WalletIndex int         `bson:"walletIndex"` //walletIndex
	Addresses   []AddressEx `bson:"addresses"`
}

type AddressEx struct { // extended
	AddressID int    `bson:"addressID"` //addressIndex
	Address   string `bson:"address"`
	Amount    int    `bson:"amount"` //float64
}

func (restClient *RestClient) addWallet() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.Split(c.GetHeader("Authorization"), " ")
		if len(authHeader) < 2 {
			log.Printf("[ERR] addWallet: wrong Authorization header len\t[addr=%s]\n", c.Request.RemoteAddr)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": msgErrHeaderError,
			})
			return
		}

		token := authHeader[1]

		var (
			code    int
			message string
		)

		var wp WalletParams

		err := decodeBody(c, &wp)
		if err != nil {
			log.Printf("[ERR] addWallet: decodeBody: %s\t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
		}

		wallet := createWallet(wp.CurrencyID, wp.Address, wp.AddressIndex, wp.WalletIndex, wp.WalletName)

		sel := bson.M{"devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets": wallet}}

		if err := restClient.userStore.Update(sel, update); err != nil {
			log.Printf("[ERR] addWallet: restClient.userStore.Update: %s\t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
			code = http.StatusBadRequest
			message = msgErrUserNotFound
		} else {
			code = http.StatusOK
			message = "wallet created"
		}

		c.JSON(http.StatusCreated, gin.H{
			"code":    code,
			"message": message,
		})
		return
	}
}

func (restClient *RestClient) addAddress() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.Split(c.GetHeader("Authorization"), " ")
		if len(authHeader) < 2 {
			log.Printf("[ERR] addAddress: wrong Authorization header len\t[addr=%s]\n", c.Request.RemoteAddr)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": msgErrHeaderError,
			})
			return
		}

		token := authHeader[1]

		var sw SelectWallet
		err := decodeBody(c, &sw)
		if err != nil {
			log.Printf("[ERR] addAddress: decodeBody: %s\t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
		}

		addr := store.Address{
			Address:      sw.Address,
			AddressIndex: sw.AddressIndex,
		}

		var (
			code    int
			message string
		)

		sel := bson.M{"wallets.walletIndex": sw.WalletIndex, "devices.JWT": token}
		update := bson.M{"$push": bson.M{"wallets.$.addresses": addr}}

		if err = restClient.userStore.Update(sel, update); err != nil {
			log.Printf("[ERR] addAddress: restClient.userStore.Update: %s\t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
			code = http.StatusBadRequest
			message = msgErrUserNotFound
		} else {
			code = http.StatusOK
			message = "address added"
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": message,
		})
	}
}

func (restClient *RestClient) getWallets() gin.HandlerFunc { //recieve rgb(255, 0, 0)
	return func(c *gin.Context) {
		var user store.User

		authHeader := strings.Split(c.GetHeader("Authorization"), " ")
		if len(authHeader) < 2 {
			log.Printf("[ERR] getWallets: wrong Authorization header len\t[addr=%s]\n", c.Request.RemoteAddr)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":        http.StatusBadRequest,
				"message":     msgErrHeaderError,
				"userWallets": user,
			})
			return
		}

		token := authHeader[1]

		var (
			code    int
			message string
		)

		sel := bson.M{"devices.JWT": token}
		if err := restClient.userStore.FindUser(sel, &user); err != nil {
			log.Printf("[ERR] getWallets: restClient.userStore.FindUser: %s\t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
			code = http.StatusBadRequest
			message = msgErrUserNotFound
		} else {
			code = http.StatusOK
			message = http.StatusText(http.StatusOK)
		}

		c.JSON(http.StatusOK, gin.H{
			"code":        code,
			"message":     message,
			"userWallets": user.Wallets,
		})

	}
}

func (restClient *RestClient) getFeeRate() gin.HandlerFunc {
	return func(c *gin.Context) {

		var rates []store.RatesRecord
		var sp EstimationSpeeds
		speeds := []int{
			1, 2, 3, 4, 5,
		}
		if err := restClient.userStore.GetAllRates("category", &rates); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"speeds":  sp,
				"code":    http.StatusInternalServerError,
				"message": msgErrRatesError,
			})
		}

		sp = EstimationSpeeds{
			VerySlow: rates[speeds[0]].Category,
			Slow:     rates[speeds[1]].Category,
			Medium:   rates[speeds[2]].Category,
			Fast:     rates[speeds[3]].Category,
			VeryFast: rates[speeds[4]].Category,
		}
		c.JSON(http.StatusOK, gin.H{
			"speeds":  sp,
			"code":    http.StatusOK,
			"message": http.StatusText(http.StatusOK),
		})

	}
}

func (restClient *RestClient) getSpendableOutputs() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.Split(c.GetHeader("Authorization"), " ")
		if len(authHeader) < 2 {
			log.Printf("[ERR] getSpendableOutputs: wrong Authorization header len\t[addr=%s]\n", c.Request.RemoteAddr)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": msgErrHeaderError,
			})
			return
		}

		token := authHeader[1]

		outWallet := store.Wallet{}

		walletIndex, err := strconv.Atoi(c.Param("walletIndex"))
		if err != nil {
			log.Printf("[ERR] getSpendableOutputs: non int wallet index: %s \t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
			c.JSON(http.StatusOK, gin.H{
				"code":    http.StatusBadRequest,
				"message": msgErrDecodeWalletIndexErr,
				"outs":    outWallet,
			})
			return
		}

		var user store.User
		var (
			code    int
			message string
		)

		sel := bson.M{"devices.JWT": token, "wallets.walletIndex": walletIndex}

		if err = restClient.userStore.FindUser(sel, &user); err != nil {
			log.Printf("[ERR] getSpendableOutputs: restClient.userStore.FindUser: %s\t[addr=%s]\n", err.Error(), c.Request.RemoteAddr)
			code = http.StatusBadRequest
			message = msgErrUserNotFound
		} else {
			code = http.StatusOK
			message = http.StatusText(http.StatusOK)
		}

	loop:
		for _, wallet := range user.Wallets {
			if wallet.WalletIndex == walletIndex {
				outWallet = wallet
				break loop
			}
		}

		if len(outWallet.Adresses) == 0 {
			message = msgErrNoSpendableOuts
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    code,
				"message": message,
				"outs":    outWallet,
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
// 				assets[name] += getaddressbalance(c, addr.Address, 1)
// 			}
// 		case currencies.String(currencies.Ether):
// 			for _, addr := range as.Adresses {
// 				assets[name] += getaddressbalance(c, addr.Address, 2)
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
