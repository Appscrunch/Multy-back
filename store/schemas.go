package store

/*
Copyright 2018 Idealnaya rabota LLC
Licensed under Multy.io license.
See LICENSE for details
*/

import (
	"time"

	"github.com/graarh/golang-socketio"
)

// Transactions statuses according to its names
const (
	TxStatusAppearedInMempoolIncoming  = 1
	TxStatusAppearedInBlockIncoming    = 2
	TxStatusAppearedInMempoolOutcoming = 3
	TxStatusAppearedInBlockOutcoming   = 4
	TxStatusInBlockConfirmedIncoming   = 5
	TxStatusInBlockConfirmedOutcoming  = 6
	TopicTransaction                   = "TransactionUpdate"
	TopicNewIncoming                   = "NewIncoming"
)

// User represents a single app user
type User struct {
	UserID  string   `bson:"userID"`  // User uqnique identifier
	Devices []Device `bson:"devices"` // All user devices
	Wallets []Wallet `bson:"wallets"` // All user addresses in all chains
}

// BTCTransaction is the way BTC transactions are storing
type BTCTransaction struct {
	Hash    string                `json:"hash"`
	Txid    string                `json:"txid"`
	Time    time.Time             `json:"time"`
	Outputs map[string]*BTCOutput `json:"outputs"` // addresses to outputs, key = address
}

// BTCOutput is the way BTC spouts are storing
type BTCOutput struct {
	Address     string  `json:"address"`
	Amount      float64 `json:"amount"`
	TxIndex     uint32  `json:"txIndex"`
	TxOutScript string  `json:"txOutScript"`
}

// TxInfo is the way TXs are stroing
type TxInfo struct {
	Type    string  `json:"type"`
	TxHash  string  `json:"txhash"`
	Address string  `json:"address"`
	Amount  float64 `json:"amount"`
}

// Device represents a single users device.
type Device struct {
	DeviceID       string `bson:"deviceID"`       // Device uqnique identifier
	PushToken      string `bson:"pushToken"`      // Firebase
	JWT            string `bson:"JWT"`            // Device JSON Web Token
	LastActionTime int64  `bson:"lastActionTime"` // Last action time from current device
	LastActionIP   string `bson:"lastActionIP"`   // IP from last session
	AppVersion     string `bson:"appVersion"`     // Mobile app verson
	DeviceType     int    `bson:"deviceType"`     // 1 - IOS, 2 - Android
}

// Wallet statuses
// ok - working one
// deleted - deleted one
// deleted will be restored is there
// is funds sent to this wallet after
// removing it
const (
	WalletStatusOK      = "ok"
	WalletStatusDeleted = "deleted"
)

// Wallet Specifies a concrete wallet of user.
type Wallet struct {
	CurrencyID     int       `bson:"currencyID"`  // Currency of wallet
	NetworkID      int       `bson:"networkID"`   // Sub-net of currency 0 - main 1 - test
	WalletIndex    int       `bson:"walletIndex"` // Wallet identifier
	WalletName     string    `bson:"walletName"`  // Wallet identifier
	LastActionTime int64     `bson:"lastActionTime"`
	DateOfCreation int64     `bson:"dateOfCreation"`
	Adresses       []Address `bson:"addresses"` // All addresses assigned to this wallet.
	Status         string    `bson:"status"`
}

// RatesRecord strores recorded rates when TX
type RatesRecord struct {
	Category int    `json:"category" bson:"category"`
	TxHash   string `json:"txHash" bson:"txHash"`
}

// Address is the way addresses are stored
type Address struct {
	AddressIndex   int    `json:"addressIndex" bson:"addressIndex"`
	Address        string `json:"address" bson:"address"`
	LastActionTime int64  `json:"lastActionTime" bson:"lastActionTime"`
}

// WalletsSelect is the way selected wallets returns
type WalletsSelect struct {
	Wallets []struct {
		Addresses []struct {
			AddressIndex int    `bson:"addressIndex"`
			Address      string `bson:"address"`
		} `bson:"addresses"`
		WalletIndex int `bson:"walletIndex"`
	} `bson:"wallets"`
}

// WalletForTx is the way wallets for TXs are stored
type WalletForTx struct {
	UserID      string           `json:"userid"`
	WalletIndex int              `json:"walletindex"`
	Address     AddressForWallet `json:"address"`
}

// AddressForWallet is the way addresses of wallet are stored
type AddressForWallet struct {
	AddressIndex    int    `json:"addressindex"`
	AddressOutIndex int    `json:"addresoutindex"`
	Address         string `json:"address"`
	Amount          int64  `json:"amount"`
}

// MultyTX is the way how user transations store in db
type MultyTX struct {
	UserID            string                `json:"userid"`
	TxID              string                `json:"txid"`
	TxHash            string                `json:"txhash"`
	TxOutScript       string                `json:"txoutscript"`
	TxAddress         []string              `json:"addresses"` //this is major addresses of the transaction (if send - inputs addresses of our user, if get - outputs addresses of our user)
	TxStatus          int                   `json:"txstatus"`
	TxOutAmount       int64                 `json:"txoutamount"`
	BlockTime         int64                 `json:"blocktime"`
	BlockHeight       int64                 `json:"blockheight"`
	Confirmations     int                   `json:"confirmations"`
	TxFee             int64                 `json:"txfee"`
	MempoolTime       int64                 `json:"mempooltime"`
	StockExchangeRate []ExchangeRatesRecord `json:"stockexchangerate"`
	TxInputs          []AddressAmount       `json:"txinputs"`
	TxOutputs         []AddressAmount       `json:"txoutputs"`
	WalletsInput      []WalletForTx         `json:"walletsinput"`  //here we storing all wallets and addresses that took part in Inputs of the transaction
	WalletsOutput     []WalletForTx         `json:"walletsoutput"` //here we storing all wallets and addresses that took part in Outputs of the transaction
}

// BTCResync is the way BTC is resyncing
type BTCResync struct {
	Txs    []MultyTX
	SpOuts []SpendableOutputs
}

// ResyncTx is the way TXs is resyncing
type ResyncTx struct {
	Hash        string
	BlockHeight int
}

// WsTxNotify is the way for tx notifying
type WsTxNotify struct {
	CurrencyID      int    `json:"currencyid"`
	NetworkID       int    `json:"networkid"`
	Address         string `json:"address"`
	Amount          string `json:"amount"`
	TxID            string `json:"txid"`
	TransactionType int    `json:"transactionType"`
	WalletIndex     int    `json:"walletindex"`
}

// TransactionWithUserID is the way getting TX without UserID
type TransactionWithUserID struct {
	NotificationMsg *WsTxNotify
	UserID          string
}

// AddressAmount is the way of getting amount of currencies on selected address
type AddressAmount struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
}

// TxRecord is the way TXs are strored
type TxRecord struct {
	UserID       string    `json:"userid"`
	Transactions []MultyTX `json:"transactions"`
}

// ExchangeRatesRecord presents record with exchanges from rate stock
// with additional information, such as date and exchange stock
type ExchangeRatesRecord struct {
	Exchanges     ExchangeRates `json:"exchanges"`
	Timestamp     int64         `json:"timestamp"`
	StockExchange string        `json:"stock_exchange"`
}

// ExchangeRates stores exchange rates
type ExchangeRates struct {
	EURtoBTC float64 `json:"eur_btc"`
	USDtoBTC float64 `json:"usd_btc"`
	ETHtoBTC float64 `json:"eth_btc"`

	ETHtoUSD float64 `json:"eth_usd"`
	ETHtoEUR float64 `json:"eth_eur"`

	BTCtoUSD float64 `json:"btc_usd"`
}

// RatesAPIBitstamp is the way rates from Bitstamp API are stored
type RatesAPIBitstamp struct {
	Date  string `json:"date"`
	Price string `json:"price"`
}

// SpendableOutputs is subentity with form avalible spendable balance
type SpendableOutputs struct {
	TxID              string                `json:"txid"`
	TxOutID           int                   `json:"txoutid"`
	TxOutAmount       int64                 `json:"txoutamount"`
	TxOutScript       string                `json:"txoutscript"`
	Address           string                `json:"address"`
	UserID            string                `json:"userid"`
	WalletIndex       int                   `json:"walletindex"`
	AddressIndex      int                   `json:"addressindex"`
	TxStatus          int                   `json:"txstatus"`
	StockExchangeRate []ExchangeRatesRecord `json:"stockexchangerate"`
}

// WalletETH is the way ETH's wallets are stored
type WalletETH struct {
	CurrencyID  int `bson:"currencyID"` // Currency of wallet
	NetworkID   int `bson:"networkID"`
	WalletIndex int `bson:"walletIndex"` //wallet identifier

	WalletName string `bson:"walletName"`

	LastActionTime int64 `bson:"lastActionTime"`

	DateOfCreation int64 `bson:"dateOfCreation"`

	// All addresses assigned to this wallet.
	Adresses []Address `bson:"addresses"`

	// Wallet status
	Status string `bson:"status"`

	// Balance of the eth wallet in wei
	Balance int64 `bson:"balance"`

	// Nonce of the wallet - index of the last transaction
	Nonce int64 `bson:"nonce"`
}

// TransactionETH is the way TXs in ETH are stored
type TransactionETH struct {
	UserID            string                `json:"userid"`
	WalletIndex       int                   `json:"walletindex"`
	AddressIndex      int                   `json:"addressindex"`
	Hash              string                `json:"txhash"`
	From              string                `json:"from"`
	To                string                `json:"to"`
	Amount            string                `json:"txoutamount"`
	GasPrice          int64                 `json:"gasprice"`
	GasLimit          int64                 `json:"gaslimit"`
	Nonce             int                   `json:"nonce"`
	Status            int                   `json:"txstatus" bson:"txstatus"`
	BlockTime         int64                 `json:"blocktime"`
	PoolTime          int64                 `json:"mempooltime"`
	BlockHeight       int64                 `json:"blockheight"`
	Confirmations     int                   `json:"confirmations"`
	StockExchangeRate []ExchangeRatesRecord `json:"stockexchangerate"`
}

// CoinType is the way coins are stotred
type CoinType struct {
	СurrencyID int `bson:"currencyID"`
	NetworkID  int `bson:"networkID"`
	GRPCUrl    string
}

// MempoolRecord is the way mempool records are stored
type MempoolRecord struct {
	Category int    `json:"category"`
	HashTX   string `json:"hashTX"`
}

// DeleteSpendableOutput is the way SpOuts are remove
type DeleteSpendableOutput struct {
	UserID  string
	TxID    string
	Address string
}

// DonationInfo is the way donation information are stored
// feature codes can be seen in /server/config
type DonationInfo struct {
	FeatureCode     int
	DonationAddress string
}

// AddressExtended is the way addresses are adding
type AddressExtended struct {
	UserID       string
	WalletIndex  int
	AddressIndex int
}

// ServerConfig contains all of the build information
type ServerConfig struct {
	BranchName string `json:"branch"`
	CommitHash string `json:"commit"`
	Build      string `json:"build_time"`
	Tag        string `json:"tag"`
}

// Donation Statuses
// 0 - Pending
// 1 - Active
// 2 - Closed
// 3 - Canceled
type Donation struct {
	FeatureID int    `json:"id"`
	Address   string `json:"address"`
	Amount    int64  `json:"amount"`
	Status    int    `json:"status"`
}

// ServiceInfo the same as ServerConfig
type ServiceInfo struct {
	Branch    string
	Commit    string
	Buildtime string
	Lasttag   string
}

// Receiver is the way information about reciever are stored
type Receiver struct {
	ID         string `json:"userid"`
	UserCode   string `json:"usercode"`
	CurrencyID int    `json:"currencyid"`
	NetworkID  int    `json:"networkid"`
	Address    string `json:"address"`
	Amount     string `json:"amount"`
	Socket     *gosocketio.Channel
}

// Sender is the way information about sender are stored
type Sender struct {
	ID       string `json:"userid"`
	UserCode string `json:"usercode"`
	Visible  map[string]bool
	Socket   *gosocketio.Channel
}

// ReceiverInData contains receiver's data
type ReceiverInData struct {
	ID         string `json:"userid"`
	CurrencyID int    `json:"currencyid"`
	Amount     int64  `json:"amount"`
	UserCode   string `json:"usercode"`
}

// SenderInData contains senders's data
type SenderInData struct {
	Code    string   `json:"usercode"`
	UserID  string   `json:"userid"`
	Visible []string `json:"visible"`
}

// PaymentData is th way to sending TX's
type PaymentData struct {
	FromID     string `json:"fromid"`
	ToID       string `json:"toid"`
	CurrencyID int    `json:"currencyid"`
	Amount     int64  `json:"amount"`
}

// NearVisible is a visible devices (see killerfeature)
type NearVisible struct {
	IDs []string `json:"ids"`
}

// RawHDTx is the way raw tx are storing
type RawHDTx struct {
	CurrencyID int    `json:"currencyid"`
	NetworkID  int    `json:"networkID"`
	UserCode   string `json:"usercode"`
	JWT        string `json:"JWT"`
	Payload    `json:"payload"`
}

// Payload is the part of transmitted data that is the actual intended message.
// Headers and metadata are sent only to enable payload delivery
type Payload struct {
	Address      string `json:"address"`
	AddressIndex int    `json:"addressindex"`
	WalletIndex  int    `json:"walletindex"`
	Transaction  string `json:"transaction"`
	IsHD         bool   `json:"ishd"`
}

// LastState gives all the informations about last block proceeding transactions
type LastState struct {
	BlockHeight int64 `bson:"blockheight"`
	CurrencyID  int   `bson:"currencyid"`
	NetworkID   int   `bson:"networkid"`
}
