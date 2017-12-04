package store

import "time"

// User represents a single app user
type User struct {
	UserID  string   `bson:"userID"`  // User uqnique identifier
	Devices []Device `bson:"devices"` // All user devices
	Wallets []Wallet `bson:"wallets"` // All user adresses in all chains
}

// Device represents a single users device.
type Device struct {
	DeviceID       string    `bson:"deviceID"`       // Device uqnique identifier (MAC adress of device)
	JWT            string    `bson:"JWT"`            // Device JSON Web Token
	LastActionTime time.Time `bson:"lastActionTime"` // Last action time from current device
	LastActionIP   string    `bson:"lastActionIP"`   // IP from last session
}

// Wallet Specifies a concrete wallet of user
type Wallet struct {
	Chain    string    `bson:"chain"`    // Currency of wallet
	Adresses []Address `bson:"adresses"` // All addresses assigned to this wallet
}

type Address struct {
	AddressID string `bson:"addressID"`
	Address   string `bson:"address"`
}
