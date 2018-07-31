/*
 * Copyright 2018 Idealnaya rabota LLC
 * Licensed under Multy.io license.
 * See LICENSE for details
 */

package store

// GetWallet gets wallet from user
// using concrete network id, currency id and wallet index
func (user *User) GetWallet(networkID, currencyID, walletIndex int) (wallet Wallet) {
	for _, w := range user.Wallets {
		if w.NetworkID == networkID && w.CurrencyID == currencyID && w.WalletIndex == walletIndex {
			wallet = w
			return
		}
	}
	return
}
