package mongodb

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/anyswap/CrossChain-Bridge/common"
	"github.com/anyswap/CrossChain-Bridge/log"
	"github.com/anyswap/CrossChain-Bridge/tokens/router"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	allChainIDs = "all"
)

func getRouterSwapKey(fromChainID, txid string, logindex int) string {
	return fmt.Sprintf("%v:%v:%v", fromChainID, txid, logindex)
}

// AddRouterSwap add router swap
func AddRouterSwap(ms *MgoSwap) error {
	ms.Key = getRouterSwapKey(ms.FromChainID, ms.TxID, ms.LogIndex)
	err := collRouterSwap.Insert(ms)
	if err == nil {
		log.Info("mongodb add router swap success", "chainid", ms.FromChainID, "txid", ms.TxID, "logindex", ms.LogIndex)
	} else {
		log.Debug("mongodb add router swap failed", "chainid", ms.FromChainID, "txid", ms.TxID, "logindex", ms.LogIndex, "err", err)
	}
	return mgoError(err)
}

// UpdateRouterSwapStatus update router swap status
func UpdateRouterSwapStatus(fromChainID, txid string, logindex int, status SwapStatus, timestamp int64, memo string) error {
	key := getRouterSwapKey(fromChainID, txid, logindex)
	updates := bson.M{"status": status, "timestamp": timestamp}
	if memo != "" {
		updates["memo"] = memo
	} else if status == TxNotSwapped || status == TxNotStable {
		updates["memo"] = ""
	}
	if status == TxNotStable {
		retryLock.Lock()
		defer retryLock.Unlock()
		swap, _ := FindRouterSwap(fromChainID, txid, logindex)
		if !(swap.Status.CanRetry() || swap.Status.CanReverify()) {
			return nil
		}
	}
	err := collRouterSwap.UpdateId(key, bson.M{"$set": updates})
	if err == nil {
		printLog := log.Info
		switch status {
		case TxVerifyFailed, TxSwapFailed:
			printLog = log.Warn
		}
		printLog("mongodb update router swap status success", "chainid", fromChainID, "txid", txid, "logindex", logindex, "status", status)
	} else {
		log.Debug("mongodb update router swap status failed", "chainid", fromChainID, "txid", txid, "logindex", logindex, "status", status, "err", err)
	}
	return mgoError(err)
}

// FindRouterSwap find router swap
func FindRouterSwap(fromChainID, txid string, logindex int) (*MgoSwap, error) {
	if logindex == 0 {
		return findFirstRouterSwap(fromChainID, txid)
	}
	key := getRouterSwapKey(fromChainID, txid, logindex)
	result := &MgoSwap{}
	err := collRouterSwap.FindId(key).One(result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

func findFirstRouterSwap(fromChainID, txid string) (*MgoSwap, error) {
	result := &MgoSwap{}
	query := getChainAndTxIDQuery(fromChainID, txid)
	err := collRouterSwap.Find(query).One(result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

func getChainAndTxIDQuery(fromChainID, txid string) bson.M {
	qtxid := bson.M{"txid": txid}
	qchainid := bson.M{"fromChainID": fromChainID}
	return bson.M{"$and": []bson.M{qtxid, qchainid}}
}

func getStatusQuery(status SwapStatus, septime int64) bson.M {
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	qstatus := bson.M{"status": status}
	queries := []bson.M{qtime, qstatus}
	return bson.M{"$and": queries}
}

func getStatusQueryWithChainID(fromChainID string, status SwapStatus, septime int64) bson.M {
	qtime := bson.M{"timestamp": bson.M{"$gte": septime}}
	qstatus := bson.M{"status": status}
	qchainid := bson.M{"fromChainID": fromChainID}
	queries := []bson.M{qtime, qstatus, qchainid}
	return bson.M{"$and": queries}
}

// FindRouterSwapsWithStatus find router swap with status
func FindRouterSwapsWithStatus(status SwapStatus, septime int64) ([]*MgoSwap, error) {
	query := getStatusQuery(status, septime)
	q := collRouterSwap.Find(query).Sort("timestamp").Limit(maxCountOfResults)
	result := make([]*MgoSwap, 0, 20)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// FindRouterSwapsWithChainIDAndStatus find router swap with chainid and status in the past septime
func FindRouterSwapsWithChainIDAndStatus(fromChainID string, status SwapStatus, septime int64) ([]*MgoSwap, error) {
	query := getStatusQueryWithChainID(fromChainID, status, septime)
	q := collRouterSwap.Find(query).Sort("timestamp").Limit(maxCountOfResults)
	result := make([]*MgoSwap, 0, 20)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// AddRouterSwapResult add router swap result
func AddRouterSwapResult(mr *MgoSwapResult) error {
	mr.Key = getRouterSwapKey(mr.FromChainID, mr.TxID, mr.LogIndex)
	err := collRouterSwapResult.Insert(mr)
	if err == nil {
		log.Info("mongodb add router swap result success", "chainid", mr.FromChainID, "txid", mr.TxID, "logindex", mr.LogIndex)
	} else {
		log.Debug("mongodb add router swap result failed", "chainid", mr.FromChainID, "txid", mr.TxID, "logindex", mr.LogIndex, "err", err)
	}
	return mgoError(err)
}

// UpdateRouterSwapResultStatus update router swap result status
func UpdateRouterSwapResultStatus(fromChainID, txid string, logindex int, status SwapStatus, timestamp int64, memo string) error {
	key := getRouterSwapKey(fromChainID, txid, logindex)
	updates := bson.M{"status": status, "timestamp": timestamp}
	if memo != "" {
		updates["memo"] = memo
	} else if status == MatchTxEmpty {
		updates["memo"] = ""
		updates["swaptx"] = ""
		updates["oldswaptxs"] = nil
		updates["swapheight"] = 0
		updates["swaptime"] = 0
	}
	err := collRouterSwapResult.UpdateId(key, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update swap result status success", "chainid", fromChainID, "txid", txid, "logindex", logindex, "status", status)
	} else {
		log.Debug("mongodb update swap result status failed", "chainid", fromChainID, "txid", txid, "logindex", logindex, "status", status, "err", err)
	}
	return mgoError(err)
}

// FindRouterSwapResult find router swap result
func FindRouterSwapResult(fromChainID, txid string, logindex int) (*MgoSwapResult, error) {
	if logindex == 0 {
		return findFirstRouterSwapResult(fromChainID, txid)
	}
	key := getRouterSwapKey(fromChainID, txid, logindex)
	result := &MgoSwapResult{}
	err := collRouterSwapResult.FindId(key).One(result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

func findFirstRouterSwapResult(fromChainID, txid string) (*MgoSwapResult, error) {
	result := &MgoSwapResult{}
	query := getChainAndTxIDQuery(fromChainID, txid)
	err := collRouterSwapResult.Find(query).One(result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// FindRouterSwapResultsWithStatus find router swap result with status
func FindRouterSwapResultsWithStatus(status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	query := getStatusQuery(status, septime)
	q := collRouterSwapResult.Find(query).Sort("timestamp").Limit(maxCountOfResults)
	result := make([]*MgoSwapResult, 0, 20)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// FindRouterSwapResultsWithChainIDAndStatus find router swap result with chainid and status in the past septime
func FindRouterSwapResultsWithChainIDAndStatus(fromChainID string, status SwapStatus, septime int64) ([]*MgoSwapResult, error) {
	query := getStatusQueryWithChainID(fromChainID, status, septime)
	q := collRouterSwapResult.Find(query).Sort("timestamp").Limit(maxCountOfResults)
	result := make([]*MgoSwapResult, 0, 20)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// FindRouterSwapResults find router swap results with chainid and address
func FindRouterSwapResults(fromChainID, address string, offset, limit int) ([]*MgoSwapResult, error) {
	var queries []bson.M

	if address != "" && address != allAddresses {
		if common.IsHexAddress(address) {
			address = strings.ToLower(address)
		}
		queries = append(queries, bson.M{"from": address})
	}

	if fromChainID != "" && fromChainID != allChainIDs {
		queries = append(queries, bson.M{"fromChainID": fromChainID})
	}

	var q *mgo.Query
	switch len(queries) {
	case 0:
		q = collRouterSwapResult.Find(nil)
	case 1:
		q = collRouterSwapResult.Find(queries[0])
	default:
		q = collRouterSwapResult.Find(bson.M{"$and": queries})
	}
	if limit >= 0 {
		q = q.Skip(offset).Limit(limit)
	} else {
		q = q.Sort("-timestamp").Skip(offset).Limit(-limit)
	}
	result := make([]*MgoSwapResult, 0, 20)
	err := q.All(&result)
	if err != nil {
		return nil, mgoError(err)
	}
	return result, nil
}

// UpdateRouterSwapResult update router swap result
func UpdateRouterSwapResult(fromChainID, txid string, logindex int, items *SwapResultUpdateItems) error {
	key := getRouterSwapKey(fromChainID, txid, logindex)
	updates := bson.M{
		"timestamp": items.Timestamp,
	}
	if items.Status != KeepStatus {
		updates["status"] = items.Status
	}
	if items.SwapTx != "" {
		updates["swaptx"] = items.SwapTx
	}
	if len(items.OldSwapTxs) != 0 {
		updates["oldswaptxs"] = items.OldSwapTxs
	}
	if items.SwapHeight != 0 {
		updates["swapheight"] = items.SwapHeight
	}
	if items.SwapTime != 0 {
		updates["swaptime"] = items.SwapTime
	}
	if items.SwapValue != "" {
		updates["swapvalue"] = items.SwapValue
	}
	if items.SwapType != 0 {
		updates["swaptype"] = items.SwapType
	}
	if items.SwapNonce != 0 {
		updates["swapnonce"] = items.SwapNonce
	}
	if items.Memo != "" {
		updates["memo"] = items.Memo
	} else if items.Status == MatchTxNotStable {
		updates["memo"] = ""
	}
	err := collRouterSwapResult.UpdateId(key, bson.M{"$set": updates})
	if err == nil {
		log.Info("mongodb update router swap result success", "chainid", fromChainID, "txid", txid, "logindex", logindex, "updates", updates)
	} else {
		log.Debug("mongodb update router swap result failed", "chainid", fromChainID, "txid", txid, "logindex", logindex, "updates", updates, "err", err)
	}
	return mgoError(err)
}

// ----------------------------- admin functions -------------------------------------

// RouterAdminPassBigValue pass big value
func RouterAdminPassBigValue(fromChainID, txid string, logIndex int) error {
	swap, err := FindRouterSwap(fromChainID, txid, logIndex)
	if err != nil {
		return err
	}
	if swap.Status != TxWithBigValue {
		return fmt.Errorf("swap status is %v, not big value status %v", swap.Status.String(), TxWithBigValue.String())
	}
	return UpdateRouterSwapStatus(fromChainID, txid, logIndex, TxNotSwapped, time.Now().Unix(), "")
}

// RouterAdminReswap reswap
func RouterAdminReswap(fromChainID, txid string, logIndex int) error {
	swap, err := FindRouterSwap(fromChainID, txid, logIndex)
	if err != nil {
		return err
	}
	if !swap.Status.CanReswap() {
		return fmt.Errorf("swap status is %v, can not reswap", swap.Status.String())
	}

	res, err := FindRouterSwapResult(fromChainID, txid, logIndex)
	if err != nil {
		return err
	}
	if !res.Status.CanReswap() {
		return fmt.Errorf("swap result status is %v, can not reswap", res.Status.String())
	}
	if res.SwapTx == "" {
		return errors.New("swap without swaptx")
	}

	resBridge := router.GetBridgeByChainID(swap.ToChainID)
	_, err = resBridge.GetTransaction(res.SwapTx)
	if err == nil && res.Status != MatchTxFailed {
		return errors.New("swaptx exist in chain or pool")
	}
	if err != nil && res.Status == MatchTxFailed {
		return errors.New("failed swaptx not exist in chain or pool")
	}

	mpcAddress := resBridge.ChainConfig.RouterMPC
	nonce, err := resBridge.GetPoolNonce(mpcAddress, "latest")
	if err != nil {
		log.Warn("get router mpc nonce failed", "address", mpcAddress)
		return err
	}
	if nonce <= res.SwapNonce {
		return errors.New("can not retry swap with lower nonce")
	}

	log.Info("[reswap] update status to TxNotSwapped", "chainid", fromChainID, "txid", txid, "logIndex", logIndex, "swaptx", res.SwapTx)

	err = UpdateRouterSwapResultStatus(fromChainID, txid, logIndex, MatchTxEmpty, time.Now().Unix(), "")
	if err != nil {
		return err
	}

	return UpdateRouterSwapStatus(fromChainID, txid, logIndex, TxNotSwapped, time.Now().Unix(), "")
}