package lbrynet

import (
	"errors"

	"github.com/lbryio/lbrytv/models"

	"github.com/lbryio/lbrytv/app/router"
	"github.com/lbryio/lbrytv/config"
	"github.com/lbryio/lbrytv/internal/monitor"
	"github.com/lbryio/lbrytv/util/wallet"

	ljsonrpc "github.com/lbryio/lbry.go/v2/extras/jsonrpc"
)

const accountNamePrefix string = "lbrytv-user-id:"
const accountNameTemplate string = accountNamePrefix + "%v"

var defaultWalletOpts = ljsonrpc.WalletCreateOpts{SkipOnStartup: false, CreateAccount: true, SingleKey: true}

var Logger = monitor.NewModuleLogger("lbrynet")

// InitializeWallet creates a wallet that can be immediately used
// in subsequent commands.
// It can recover from errors like existing wallets, but if a wallet is known to exist
// (eg. a wallet ID stored in the database already), AddWallet should be called instead.
func InitializeWallet(uid int) (models.LbrynetServer, string, error) {
	wid := wallet.MakeID(uid)
	log := Logger.LogF(monitor.F{"wallet_id": wid, "user_id": uid})
	wallet, lbrynetServer, err := CreateWallet(uid)
	if err != nil {
		if errors.As(err, &WalletExists{}) {
			log.Warn(err.Error())
			return lbrynetServer, wid, nil
		} else if errors.As(err, &WalletNeedsLoading{}) {
			log.Info(err.Error())
			wallet, err = AddWallet(uid)
			if err != nil && errors.As(err, &WalletAlreadyLoaded{}) {
				log.Info(err.Error())
				return lbrynetServer, wid, nil
			}
		} else {
			log.Error("don't know how to recover from error: ", err)
			return lbrynetServer, "", err
		}
	}
	return lbrynetServer, wallet.ID, nil
}

// CreateWallet creates a new wallet with the LbrynetServer.
// Returned error doesn't necessarily mean that the wallet is not operational:
//
// 	if errors.Is(err, lbrynet.WalletExists) {
// 	 // Okay to proceed with the account
//  }
//
// 	if errors.Is(err, lbrynet.WalletNeedsLoading) {
// 	 // AddWallet() needs to be called before the wallet can be used
//  }
func CreateWallet(uid int) (*ljsonrpc.Wallet, models.LbrynetServer, error) {
	lbrynetRouter := router.New(config.GetLbrynetServers())
	wid := wallet.MakeID(uid)
	log := Logger.LogF(monitor.F{"wallet_id": wid, "user_id": uid})
	lbrynetServer := lbrynetRouter.GetSDKServer(wid)
	client := ljsonrpc.NewClient(lbrynetServer.Address)
	wallet, err := client.WalletCreate(wid, &defaultWalletOpts)
	if err != nil {
		return nil, lbrynetServer, NewWalletError(uid, err)
	}
	log.Info("wallet created")
	return wallet, lbrynetServer, nil
}

// AddWallet loads an existing wallet in the LbrynetServer.
// May return errors:
//  WalletAlreadyLoaded - wallet is already loaded and operational
//  WalletNotFound - wallet file does not exist and won't be loaded.
func AddWallet(uid int) (*ljsonrpc.Wallet, error) {
	lbrynetRouter := router.New(config.GetLbrynetServers())
	wid := wallet.MakeID(uid)
	log := Logger.LogF(monitor.F{"wallet_id": wid, "user_id": uid})
	client := ljsonrpc.NewClient(lbrynetRouter.GetSDKServerAddress(wid))
	wallet, err := client.WalletAdd(wid)
	if err != nil {
		return nil, NewWalletError(uid, err)
	}
	log.Info("wallet loaded")
	return wallet, nil
}

// WalletRemove loads an existing wallet in the LbrynetServer.
// May return errors:
//  WalletAlreadyLoaded - wallet is already loaded and operational
//  WalletNotFound - wallet file does not exist and won't be loaded.
func WalletRemove(uid int) (*ljsonrpc.Wallet, error) {
	lbrynetRouter := router.New(config.GetLbrynetServers())
	wid := wallet.MakeID(uid)
	log := Logger.LogF(monitor.F{"wallet_id": wid, "user_id": uid})
	client := ljsonrpc.NewClient(lbrynetRouter.GetSDKServerAddress(wid))
	wallet, err := client.WalletRemove(wid)
	if err != nil {
		return nil, NewWalletError(uid, err)
	}
	log.Info("wallet removed")
	return wallet, nil
}

// Resolve calls resolve method on the daemon and handles
// *frequent* LbrynetServer response format changes with grace instead of panicking.
func Resolve(url string) (*ljsonrpc.Claim, error) {
	lbrynetRouter := router.New(config.GetLbrynetServers())
	// Client is a LBRY LbrynetServer jsonrpc client instance
	var Client = ljsonrpc.NewClient(lbrynetRouter.GetBalancedSDKAddress())
	r, err := Client.Resolve(url)
	if err != nil {
		return nil, err
	}
	item := (*r)[url]

	if item.CanonicalURL == "" {
		return nil, errors.New("invalid resolve response structure from sdk client")
	}
	return &item, nil
}
