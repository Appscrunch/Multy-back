/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package eos

import (
	"context"
	"fmt"
	"strings"

	"github.com/Multy-io/Multy-EOS-node-service/proto"
	"github.com/Multy-io/Multy-back/store"
	"github.com/pkg/errors"
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
	OwnerKey       string           `json:"ownerkey"`
	ActiveKey      string           `json:"activekey"`
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
		if err != nil {
			// skip address log error
			log.Errorf("GetTokenBalance(%s): %s", addr.Address, err)
			continue
		}
		if len(balance.Assets) == 0 {
			log.Errorf("GetTokenBalance(%s): no assets", addr.Address)
			continue
		}
		var amount int64
		for _, asset := range balance.Assets {
			if asset.Symbol == "EOS" {
				amount = asset.GetAmount()
			}
		}
		balances = append(balances, AddressBalance{
			Amount:       amount,
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

// TotalBalance gets balance of all the addresses and presents it in string
func TotalBalance(balances []AddressBalance) string {
	var totalAmount int64
	for _, balance := range balances {
		totalAmount += balance.Amount
	}

	result := fmt.Sprintf("%d", totalAmount)

	return result
}

func (conn *Conn) GetWalleVerbose(ctx context.Context, wallet store.Wallet) (res Wallet, err error) {
	balances, err := conn.GetBalance(ctx, wallet)
	if err != nil {
		return res, errors.Wrap(err, "eos.GetBalance")
	}

	var ownerKey, activeKey string
	if len(wallet.Adresses) > 0 {
		info, err := conn.Client.AccountCheck(ctx, &proto.Account{Name: wallet.Adresses[0].Address})
		if err != nil {
			return res, errors.Wrap(err, "eos.AccountCheck")
		}
		ownerKey, activeKey = info.OwnerKey, info.ActiveKey
	}

	res = Wallet{
		LastActionTime: wallet.LastActionTime,
		WalletIndex:    wallet.WalletIndex,
		NetworkID:      wallet.NetworkID,
		CurrencyID:     wallet.CurrencyID,
		WalletName:     wallet.WalletName,
		DateOfCreation: wallet.DateOfCreation,
		VerboseAddress: balances,
		Balance:        TotalBalance(balances),
		OwnerKey:       ownerKey,
		ActiveKey:      activeKey,
		// TODO make pending based on irreversible block num
		Pending:        false,
		PendingBalance: "0",
	}
	return res, nil
}
