/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package eos

import (
	"context"
	"fmt"
	"github.com/Multy-io/Multy-EOS-node-service/proto"
	"github.com/Multy-io/Multy-back/store"
	"strings"
)

type Wallet struct {
	CurrencyID     int              `json:"currencyid"`
	NetworkID      int              `json:"networkid"`
	WalletIndex    int              `json:"walletindex"`
	WalletName     string           `json:"walletname"`
	LastActionTime int64            `json:"lastactiontime"`
	DateOfCreation int64            `json:"dateofcreation"`
	VerboseAddress []AddressBalance `json:"addresses"`
	PendingBalance string           `json:"pendingbalance"`
	Balance        string           `json:"balance"`
	Pending        bool             `json:"pending"`
}

type AddressBalance struct {
	Amount       int64  `json:"amount"`
	Address      string `json:"address"`
	AddressIndex int    `json:"addressindex"`
}

func (conn *Conn) GetBalance(ctx context.Context, wallet store.Wallet) ([]AddressBalance, error) {
	if len(wallet.Adresses) == 0 {
		return nil, fmt.Errorf("wallet has no addresses")
	}
	balances := make([]AddressBalance, 0, len(wallet.Adresses))
	for _, addr := range wallet.Adresses {
		balance, err := conn.Client.GetTokenBalance(ctx, &proto.BalanceReq{
			Account: addr.Address,
			Symbol:  "EOS",
		})
		//balance, err := conn.Client.GetAddressBalance(ctx, &proto.Account{
		//	Name: addr.Address,
		//})
		if err != nil {
			// skip address log error
			log.Errorf("GetAddressBalance(%s): %s", addr.Address, err)
			continue
		}
		balances = append(balances, AddressBalance{
			Amount:       balance.Assets[0].Amount,
			Address:      addr.Address,
			AddressIndex: addr.AddressIndex,
		})
	}
	return balances, nil
}

// ValidatePublicKey validates eos public key
func ValidatePublicKey(key string) error {
	if len(key) < 8 { // based on eos-go public key validation
		return fmt.Errorf("wrong len %d", len(key))
	}
	if !strings.HasPrefix(key, "EOS") {
		return fmt.Errorf("wrong prefix: \"%s\"", key[:3])
	}
	return nil
}
