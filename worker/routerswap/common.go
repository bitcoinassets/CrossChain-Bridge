package routerswap

import (
	"time"

	"github.com/anyswap/CrossChain-Bridge/mongodb"
	"github.com/anyswap/CrossChain-Bridge/tokens"
	"github.com/anyswap/CrossChain-Bridge/tokens/router"
)

// MatchTx struct
type MatchTx struct {
	SwapTx     string
	OldSwapTxs []string
	SwapHeight uint64
	SwapTime   uint64
	SwapValue  string
	SwapType   tokens.SwapType
	SwapNonce  uint64
}

func addInitialSwapResult(swapInfo *tokens.TxSwapInfo, status mongodb.SwapStatus) (err error) {
	txid := swapInfo.Hash
	swapType := tokens.RouterSwapType
	swapResult := &mongodb.MgoSwapResult{
		PairID:        swapInfo.PairID,
		TxID:          txid,
		TxTo:          swapInfo.TxTo,
		TxHeight:      swapInfo.Height,
		TxTime:        swapInfo.Timestamp,
		From:          swapInfo.From,
		To:            swapInfo.To,
		Bind:          swapInfo.Bind,
		Value:         swapInfo.Value.String(),
		SwapTx:        "",
		SwapHeight:    0,
		SwapTime:      0,
		SwapValue:     "0",
		SwapType:      uint32(swapType),
		SwapNonce:     0,
		Status:        status,
		Timestamp:     now(),
		Memo:          "",
		ForNative:     swapInfo.ForNative,
		ForUnderlying: swapInfo.ForUnderlying,
		Token:         swapInfo.Token,
		Path:          swapInfo.Path,
		AmountOutMin:  swapInfo.AmountOutMin.String(),
		FromChainID:   swapInfo.FromChainID.String(),
		ToChainID:     swapInfo.ToChainID.String(),
		LogIndex:      swapInfo.LogIndex,
	}
	err = mongodb.AddRouterSwapResult(swapResult)
	if err != nil {
		logWorkerError("add", "addInitialSwapResult failed", err, "chainid", swapInfo.FromChainID, "txid", txid, "logIndex", swapInfo.LogIndex)
	} else {
		logWorker("add", "addInitialSwapResult success", "chainid", swapInfo.FromChainID, "txid", txid, "logIndex", swapInfo.LogIndex)
	}
	return err
}

func updateRouterSwapResult(fromChainID, txid string, logIndex int, mtx *MatchTx) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:    mongodb.MatchTxNotStable,
		Timestamp: now(),
	}
	if mtx.SwapHeight == 0 {
		updates.SwapTx = mtx.SwapTx
		updates.OldSwapTxs = mtx.OldSwapTxs
		updates.SwapValue = mtx.SwapValue
		updates.SwapNonce = mtx.SwapNonce
		updates.SwapHeight = 0
		updates.SwapTime = 0
	} else {
		updates.SwapHeight = mtx.SwapHeight
		updates.SwapTime = mtx.SwapTime
		if mtx.SwapTx != "" {
			updates.SwapTx = mtx.SwapTx
		}
	}
	err = mongodb.UpdateRouterSwapResult(fromChainID, txid, logIndex, updates)
	if err != nil {
		logWorkerError("update", "updateSwapResult failed", err,
			"chainid", fromChainID, "txid", txid, "logIndex", logIndex,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swapnonce", mtx.SwapNonce)
	} else {
		logWorker("update", "updateSwapResult success",
			"chainid", fromChainID, "txid", txid, "logIndex", logIndex,
			"swaptx", mtx.SwapTx, "swapheight", mtx.SwapHeight,
			"swaptime", mtx.SwapTime, "swapvalue", mtx.SwapValue,
			"swapnonce", mtx.SwapNonce)
	}
	return err
}

func updateSwapTx(fromChainID, txid string, logIndex int, swapTx string) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		SwapTx:    swapTx,
		Timestamp: now(),
	}
	err = mongodb.UpdateRouterSwapResult(fromChainID, txid, logIndex, updates)
	if err != nil {
		logWorkerError("update", "updateSwapTx failed", err, "chainid", fromChainID, "txid", txid, "logIndex", logIndex, "swaptx", swapTx)
	} else {
		logWorker("update", "updateSwapTx success", "chainid", fromChainID, "txid", txid, "logIndex", logIndex, "swaptx", swapTx)
	}
	return err
}

func updateOldSwapTxs(fromChainID, txid string, logIndex int, oldSwapTxs []string) (err error) {
	updates := &mongodb.SwapResultUpdateItems{
		Status:     mongodb.KeepStatus,
		OldSwapTxs: oldSwapTxs,
		Timestamp:  now(),
	}
	err = mongodb.UpdateRouterSwapResult(fromChainID, txid, logIndex, updates)
	if err != nil {
		logWorkerError("update", "updateOldSwapTxs fialed", err, "chainid", fromChainID, "txid", txid, "logIndex", logIndex, "swaptxs", len(oldSwapTxs))
	} else {
		logWorker("update", "updateOldSwapTxs success", "chainid", fromChainID, "txid", txid, "logIndex", logIndex, "swaptxs", len(oldSwapTxs))
	}
	return err
}

func markSwapResultStable(fromChainID, txid string, logIndex int) (err error) {
	status := mongodb.MatchTxStable
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateRouterSwapResultStatus(fromChainID, txid, logIndex, status, timestamp, memo)
	if err != nil {
		logWorkerError("stable", "markSwapResultStable failed", err, "chainid", fromChainID, "txid", txid, "logIndex", logIndex)
	} else {
		logWorker("stable", "markSwapResultStable success", "chainid", fromChainID, "txid", txid, "logIndex", logIndex)
	}
	return err
}

func markSwapResultFailed(fromChainID, txid string, logIndex int) (err error) {
	status := mongodb.MatchTxFailed
	timestamp := now()
	memo := "" // unchange
	err = mongodb.UpdateRouterSwapResultStatus(fromChainID, txid, logIndex, status, timestamp, memo)
	if err != nil {
		logWorkerError("stable", "markSwapResultFailed failed", err, "chainid", fromChainID, "txid", txid, "logIndex", logIndex)
	} else {
		logWorker("stable", "markSwapResultFailed success", "chainid", fromChainID, "txid", txid, "logIndex", logIndex)
	}
	return err
}

func dcrmSignTransaction(bridge *router.Bridge, rawTx interface{}, args *tokens.BuildTxArgs) (signedTx interface{}, txHash string, err error) {
	maxRetryDcrmSignCount := 5
	for i := 0; i < maxRetryDcrmSignCount; i++ {
		signedTx, txHash, err = bridge.DcrmSignTransaction(rawTx, args.GetExtraArgs())
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, "", err
	}
	return signedTx, txHash, nil
}

func sendSignedTransaction(bridge *router.Bridge, signedTx interface{}, args *tokens.BuildTxArgs, isReplace bool) (err error) {
	var (
		txHash              string
		retrySendTxCount    = 3
		retrySendTxInterval = 1 * time.Second
	)
	for i := 0; i < retrySendTxCount; i++ {
		txHash, err = bridge.SendTransaction(signedTx)
		if txHash != "" {
			if tx, _ := bridge.GetTransaction(txHash); tx != nil {
				logWorker("sendtx", "send tx success", "txHash", txHash)
				err = nil
				break
			}
		}
		time.Sleep(retrySendTxInterval)
	}
	if err != nil {
		fromChainID, txid, logIndex := args.FromChainID.String(), args.SwapID, args.LogIndex
		_ = mongodb.UpdateRouterSwapStatus(fromChainID, txid, logIndex, mongodb.TxSwapFailed, now(), err.Error())
		_ = mongodb.UpdateRouterSwapResultStatus(fromChainID, txid, logIndex, mongodb.TxSwapFailed, now(), err.Error())
		logWorkerError("sendtx", "update router swap status to TxSwapFailed", err, "txid", txid)
		return err
	}
	if !isReplace {
		bridge.IncreaseNonce(args.From, 1)
	}
	return nil
}