package multyback

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/btcsuite/btcd/rpcclient"
)

var Cert = `-----BEGIN CERTIFICATE-----
MIICPDCCAZ2gAwIBAgIQf8XOycg2EQ8wHpXsZJSy7jAKBggqhkjOPQQDBDAjMREw
DwYDVQQKEwhnZW5jZXJ0czEOMAwGA1UEAxMFYW50b24wHhcNMTcxMTI2MTY1ODQ0
WhcNMjcxMTI1MTY1ODQ0WjAjMREwDwYDVQQKEwhnZW5jZXJ0czEOMAwGA1UEAxMF
YW50b24wgZswEAYHKoZIzj0CAQYFK4EEACMDgYYABAGuHzCFKsJwlFwmtx5QMT/r
YJ/ap9E2QlUsCnMUCn1ho0wLJkpIgNQWs1zcaKTMGZNpwwLemCHke9sX06h/MdAG
CwGf1CY5kafyl7dTTlmD10sBA7UD1RXDjYnmYQhB1Z1MUNXKWXe4jCv7DnWmFEnc
+s5N1NXJx1PNzx/EcsCkRJcMraNwMG4wDgYDVR0PAQH/BAQDAgKkMA8GA1UdEwEB
/wQFMAMBAf8wSwYDVR0RBEQwQoIFYW50b26CCWxvY2FsaG9zdIcEfwAAAYcQAAAA
AAAAAAAAAAAAAAAAAYcEwKgAeYcQ/oAAAAAAAAByhcL//jB99jAKBggqhkjOPQQD
BAOBjAAwgYgCQgCfs9tYHA1nvU5HSdNeHSZCR1WziHYuZHmGE7eqAWQjypnVbFi4
pccvzDFvESf8DG4FVymK4E2T/RFnD9qUDiMzPQJCATkCMzSKcyYlsL7t1ZgQLwAK
UpQl3TYp8uTf+UWzBz0uoEbB4CFeE2G5ZzrVK4XWZK615sfVFSorxHOOZaLwZEEL
-----END CERTIFICATE-----`

var connCfg = &rpcclient.ConnConfig{
	Host:         "192.168.0.121:18334",
	User:         "multy",
	Pass:         "multy",
	Endpoint:     "ws",
	HTTPPostMode: true,  // Bitcoin core only supports HTTP POST mode
	DisableTLS:   false, // Bitcoin core does not provide TLS by default
	Certificates: []byte(Cert),
}

// TODO: add yaml tags to structures

// JWTConf is struct with JWT parameters
type JWTConf struct {
	Secret      string
	ElapsedDays int
}

// Configuration is a struct with all service options
type Configuration struct {
	Name             string
	Address          string
	DataStore        map[string]interface{}
	JWTConfig        *JWTConf
	DataStoreAddress string
}

// GetConfig initializes configuration for multy backend
func GetConfig(confiFile string) (*Configuration, error) {
	fd, err := os.Open(confiFile)
	if err != nil {
		return nil, err
	}
	rawData, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	conf := &Configuration{}
	err = json.Unmarshal([]byte(rawData), conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
