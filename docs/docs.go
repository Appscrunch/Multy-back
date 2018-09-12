// GENERATED BY THE COMMAND ABOVE; DO NOT EDIT
// This file was generated by swaggo/swag at
// 2018-09-06 18:15:05.618875 +0300 +03 m=+0.060445799

package docs

import (
	"github.com/swaggo/swag"
)

var doc = `{
    "swagger": "2.0",
    "info": {
       "version": "1.0.0",
       "title": "Multy-Back",
       "description": "Description of Multy-Back api. The continuation of https://github.com/Multy-io/Multy-back/wiki."
    },
    "securityDefinitions": {
       "Bearer": {
          "type": "apiKey",
          "name": "Authorization",
          "in": "header"
       }
    },
    "paths": {
       "/server/config": {
          "get": {
             "schemes": [
                "http"
             ],
             "summary": "Get server config",
             "responses": {
                "200": {
                   "description": "{\"android\":{\"hard\":7,\"soft\":7},\"api\":\"0.01\",\"donate\":[{\"FeatureCode\":10000,\"DonationAddress\":\"1FPv9f8EGRDNod7mSvJnUFonUioA3Pw5ng\"},{\"FeatureCode\":20409,\"DonationAddress\":\"1KbRaapwNnTJMZcpbTQfb7xJnymKFJDyC1\"}],\"ios\":{\"hard\":29,\"soft\":29},\"servertime\":1530269261,\"stockexchanges\":{\"gdax\":[\"eur_btc\",\"usd_btc\",\"eth_btc\",\"eth_usd\",\"eth_eur\",\"btc_usd\"],\"poloniex\":[\"usd_btc\",\"eth_btc\",\"eth_usd\",\"btc_usd\"]},\"version\":{\"branch\":\"release_1.1\",\"commit\":\"161a198\",\"build_time\":\"2018-06-22T15:57:34+0300\",\"tag\":\"\"}}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/donations": {
          "get": {
             "schemes": [
                "http"
             ],
             "summary": "Get donations",
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"donations\":[{\"id\":10000,\"address\":\"1FPv9f8EGRDNod7mSvJnUFonUioA3Pw5ng\",\"amount\":0,\"status\":1},{\"id\":10100,\"address\":\"1EyoGcD8sWzHikPPiJmzGiHyUyTHZzHtGU\",\"amount\":0,\"status\":1}],\"message\":\"OK\"}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/wallets/verbose": {
          "get": {
             "summary": "Get all wallets verboses",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"message\":\"OK\",\"topindexes\":[{\"currencyid\":0,\"networkid\":0,\"topindex\":2}],\"wallets\":[{\"currencyid\":0,\"networkid\":0,\"walletindex\":0,\"walletname\":\"name1\",\"lastactiontime\":1536133839,\"dateofcreation\":1536133839,\"addresses\":[{\"lastactiontime\":1536133839,\"address\":\"address\",\"addressindex\":0,\"amount\":0,\"issyncing\":false}],\"pending\":false},{\"currencyid\":0,\"networkid\":0,\"walletindex\":1,\"walletname\":\"name\",\"lastactiontime\":1536150232,\"dateofcreation\":1536150232,\"addresses\":[{\"lastactiontime\":1536150232,\"address\":\"address\",\"addressindex\":0,\"amount\":0,\"issyncing\":false}],\"pending\":false}]}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/multisig/estimate/{contractAddress}": {
          "get": {
             "summary": "Multisig Estimation",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "path",
                   "name": "contractAddress",
                   "type": "string",
                   "required": true,
                   "description": "Contract address"
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"confirmTransaction\":400000,\"deployMultisig\":5000000,\"priceOfCreation\":100000000000000000,\"revokeConfirmation\":400000,\"submitTransaction\":400000}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/transaction/feerate/{currencyID}/{networkID}": {
          "get": {
             "summary": "Get fee rate of selected currency",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "path",
                   "name": "currencyID",
                   "type": "integer",
                   "required": true
                },
                {
                   "in": "path",
                   "name": "networkID",
                   "type": "integer",
                   "required": true
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"message\":\"OK\",\"speeds\":{\"VerySlow\":2,\"Slow\":2,\"Medium\":3,\"Fast\":5,\"VeryFast\":10}}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/outputs/spendable/{currencyID}/{networkID}/{address}": {
          "get": {
             "summary": "Get spendable outputs of selected address",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "path",
                   "name": "currencyID",
                   "type": "integer",
                   "required": true,
                   "description": "Currency ID of the wallet"
                },
                {
                   "in": "path",
                   "name": "networkID",
                   "type": "integer",
                   "required": true,
                   "description": "Network ID of the wallet"
                },
                {
                   "in": "path",
                   "name": "address",
                   "type": "string",
                   "required": true,
                   "description": "Address of selected wallet"
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"message\":\"OK\",\"outs\":[{\"txid\":\"4745ad96ec7fe551f693b63982764bc284afba077f19180c64efd757a7658786\",\"txoutid\":0,\"txoutamount\":449218,\"txoutscript\":\"76a914ec4814cecf2cc4d2b87d02456928a26ccde9b66788ac\"},{\"txid\":\"647f606017217d781c34fc0b98f85846c9eade1b315b28ad3ebe781df427e8f8\",\"txoutid\":0,\"txoutamount\":112304,\"txoutscript\":\"76a914ec4814cecf2cc4d2b87d02456928a26ccde9b66788ac\"}]}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/wallet/{walletIndex}/verbose/{currencyID}/{networkID}": {
          "get": {
             "summary": "Get selected wallet verbose",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "path",
                   "name": "currencyID",
                   "type": "integer",
                   "required": true,
                   "description": "Currency ID of the wallet"
                },
                {
                   "in": "path",
                   "name": "networkID",
                   "type": "integer",
                   "required": true,
                   "description": "Network ID of the wallet"
                },
                {
                   "in": "path",
                   "name": "walletIndex",
                   "type": "integer",
                   "required": true,
                   "description": "Address of selected wallet"
                }
             ],
             "responses": {
                "200": {
                   "description": "OK"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/wallets/transactions/{currencyID}/{networkID}/{walletIndex}": {
          "get": {
             "summary": "Get selected wallet transactions",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "path",
                   "name": "currencyID",
                   "type": "integer",
                   "required": true,
                   "description": "Currency ID of the wallet"
                },
                {
                   "in": "path",
                   "name": "networkID",
                   "type": "integer",
                   "required": true,
                   "description": "Network ID of the wallet"
                },
                {
                   "in": "path",
                   "name": "walletIndex",
                   "type": "integer",
                   "required": true,
                   "description": "Address of selected wallet"
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"message\":\"OK\",\"wallet\":[{\"currencyid\":60,\"networkid\":4,\"walletindex\":0,\"walletname\":\"kek\",\"lastactiontime\":1530951267,\"dateofcreation\":1530951267,\"nonce\":1,\"pendingbalance\":\"0\",\"balance\":\"320000000000000000\",\"addresses\":[{\"lastactiontime\":1530951267,\"address\":\"0xb8fbca57279d3958ff41116d4df8ebc59e10e377\",\"addressindex\":0,\"amount\":\"320000000000000000\",\"nonce\":1}],\"pending\":false,\"multisig\":{\"owners\":[{\"UserID\":\"009f7a685960f1a8ffde85cd2c004360581aff0d97a62342c2e97c7b1e15df3fc3\",\"Address\":\"0xfad9edb6094fc4909c6f1b236ca4dd77c1165f53\",\"Associated\":true,\"WalletIndex\":0,\"AddressIndex\":0},{\"UserID\":\"\",\"Address\":\"0x8c473062eca633f47c4f9a5666f18f435f4bb221\",\"Associated\":false,\"WalletIndex\":0,\"AddressIndex\":0}],\"Confirmations\":2,\"isdeployed\":true,\"factoryaddress\":\"0x116ffa11dd8829524767f561da5d33d3d170e17d\",\"txofcreation\":\"0x7ab930d9fb29e272436600b86c94392705b979c8b2944eb28597a5e5d9111e3a\"}}]}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/auth": {
          "post": {
             "schemes": [
                "http"
             ],
             "summary": "Authentification",
             "parameters": [
                {
                   "in": "body",
                   "name": "auth",
                   "description": "Authentification",
                   "schema": {
                      "type": "object",
                      "required": [
                         "userID",
                         "deviceID",
                         "deviceType",
                         "pushToken",
                         "appversion"
                      ],
                      "properties": {
                         "userID": {
                            "type": "string"
                         },
                         "deviceID": {
                            "type": "string"
                         },
                         "deviceType": {
                            "type": "integer"
                         },
                         "pushToken": {
                            "type": "string"
                         },
                         "appversion": {
                            "type": "string"
                         }
                      }
                   }
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"expire\":\"2018-09-05T17:57:23+02:00\",\"token\":\"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1MzYxNjMwNDMsImlkIjoiU3dhZ2dlclVzZXIiLCJvcmlnX2lhdCI6MTUzNjE1OTQ0M30.QaW8xuSIjp4rqzTWPWO9_zGJ0aRxMSI36uO34hSTLr0\"}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/wallet": {
          "post": {
             "summary": "Create wallet",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "body",
                   "name": "wallet",
                   "description": "The wallet",
                   "schema": {
                      "type": "object",
                      "required": [
                         "currencyID",
                         "networkID",
                         "address",
                         "addressIndex",
                         "walletIndex",
                         "walletName"
                      ],
                      "properties": {
                         "currencyID": {
                            "type": "integer"
                         },
                         "networkID": {
                            "type": "integer"
                         },
                         "address": {
                            "type": "string"
                         },
                         "addressIndex": {
                            "type": "integer"
                         },
                         "walletIndex": {
                            "type": "integer"
                         },
                         "walletName": {
                            "type": "string"
                         },
                         "multisig": {
                            "type": "string"
                         }
                      }
                   }
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"message\":\"OK\",\"time\":1536159589}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/address": {
          "post": {
             "summary": "Add address to wallet",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "body",
                   "name": "wallet",
                   "description": "The wallet to create.",
                   "schema": {
                      "type": "object",
                      "required": [
                         "currencyID",
                         "networkID",
                         "addressIndex",
                         "walletIndex"
                      ],
                      "properties": {
                         "currencyID": {
                            "type": "integer"
                         },
                         "networkID": {
                            "type": "integer"
                         },
                         "address": {
                            "type": "string"
                         },
                         "addressIndex": {
                            "type": "integer"
                         },
                         "walletIndex": {
                            "type": "integer"
                         }
                      }
                   }
                }
             ],
             "responses": {
                "200": {
                   "description": "OK"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/wallet/name": {
          "post": {
             "summary": "Change wallet name",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "body",
                   "name": "wallet",
                   "description": "The wallet",
                   "schema": {
                      "type": "object",
                      "required": [
                         "walletname",
                         "currencyID",
                         "walletIndex",
                         "NetworkID"
                      ],
                      "properties": {
                         "walletname": {
                            "type": "string"
                         },
                         "currencyID": {
                            "type": "integer"
                         },
                         "walletIndex": {
                            "type": "integer"
                         },
                         "NetworkID": {
                            "type": "integer"
                         }
                      }
                   }
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\": \"200\", message\": \"OK\"}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/transaction/send": {
          "post": {
             "summary": "Send raw transaction",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "body",
                   "name": "wallet",
                   "description": "The wallet to create.",
                   "schema": {
                      "type": "object",
                      "required": [
                         "currencyID",
                         "networkID",
                         "payload"
                      ],
                      "properties": {
                         "currencyID": {
                            "type": "integer"
                         },
                         "networkID": {
                            "type": "integer"
                         },
                         "payload": {
                            "type": "object"
                         }
                      }
                   }
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\": 200, \"message\": \"sent\", \"txid\":\"76a914ec4814cecf2cc4d2b87d02456928a26ccde9b66788ac\"}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       },
       "/api/v1/wallet/{currencyID}/{networkID}/{walletIndex}": {
          "delete": {
             "summary": "Delete wallet",
             "security": [
                {
                   "Bearer": []
                }
             ],
             "parameters": [
                {
                   "in": "path",
                   "name": "currencyID",
                   "type": "integer",
                   "required": true,
                   "description": "Currency ID of the wallet to delete"
                },
                {
                   "in": "path",
                   "name": "networkID",
                   "type": "integer",
                   "required": true,
                   "description": "Network ID of the wallet to delete"
                },
                {
                   "in": "path",
                   "name": "walletIndex",
                   "type": "integer",
                   "required": true,
                   "description": "Wallet Index of the wallet to delete"
                }
             ],
             "responses": {
                "200": {
                   "description": "{\"code\":200,\"message\":\"OK\"}"
                },
                "400": {
                   "description": "Error code"
                }
             }
          }
       }
    },
    "host": "test.multy.io",
    "basePath": "/",
    "schemes": [
       "http"
    ]
 }`

type s struct{}

func (s *s) ReadDoc() string {
	return doc
}
func init() {
	swag.Register(swag.Name, &s{})
}
