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
)

type Balance struct {
	CurrencyID     int            `json:"currency_id"`
	NetworkID      int            `json:"network_id"`
	WalletIndex    int            `json:"wallet_index"`
	WalletName     string         `json:"wallet_name"`
	LastActionTime int64          `json:"last_action_time"`
	DateOfCreation int64          `json:"date_of_creation"`
	Assets         []*proto.Asset `json:"assets"`
}

func (conn *Conn) GetBalance(ctx context.Context, wallet store.Wallet) ([]Balance, error) {
	if len(wallet.Adresses) == 0 {
		return nil, fmt.Errorf("wallet has no addresses")
	}
	balances := make([]Balance, 0, len(wallet.Adresses))
	for _, addr := range wallet.Adresses {
		// get EOS token for now
		balance, err := conn.Client.GetTokenBalance(ctx, &proto.BalanceReq{
			Account: addr.Address,
			Symbol:  "EOS",
		})
		if err != nil {
			// skip address log error
			log.Errorf("GetTokenBalance(%s): %s", addr.Address, err)
			continue
		}
		balances = append(balances, Balance{
			Assets:         balance.Assets,
			WalletIndex:    wallet.WalletIndex,
			CurrencyID:     wallet.CurrencyID,
			NetworkID:      wallet.NetworkID,
			DateOfCreation: wallet.DateOfCreation,
			LastActionTime: wallet.LastActionTime,
			WalletName:     wallet.WalletName,
		})
	}
	return balances, nil
}
