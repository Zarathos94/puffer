package utils

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/Zarathos94/puffer/cache"
	"github.com/Zarathos94/puffer/models"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// FormatETH converts a big.Int value in wei to a human-readable ETH string with 6 decimals and K/M/B suffixes for large values.
func FormatETH(val *big.Int) string {
	if val == nil {
		return "0"
	}
	f := new(big.Float).SetInt(val)
	eth := new(big.Float).Quo(f, big.NewFloat(1e18))

	// Get float64 value for suffix logic
	ethFloat, _ := eth.Float64()
	absEth := ethFloat
	if absEth < 0 {
		absEth = -absEth
	}

	switch {
	case absEth >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", ethFloat/1_000_000_000)
	case absEth >= 1_000_000:
		return fmt.Sprintf("%.2fM", ethFloat/1_000_000)
	case absEth >= 1_000:
		return fmt.Sprintf("%.2fK", ethFloat/1_000)
	default:
		return fmt.Sprintf("%.6f", ethFloat)
	}
}

const (
	vaultAddress = "0xD9A442856C234a39a81a089C06451EBAa4306a72"
	abiJSON      = `[ { "inputs": [], "name": "totalAssets", "outputs": [ { "internalType": "uint256", "name": "", "type": "uint256" } ], "stateMutability": "view", "type": "function" }, { "inputs": [], "name": "totalSupply", "outputs": [ { "internalType": "uint256", "name": "", "type": "uint256" } ], "stateMutability": "view", "type": "function" } ]`
)

type RateService struct {
	client    *ethclient.Client
	parsedABI abi.ABI
	vault     common.Address
	cache     *cache.Cache
}

// ERC1967 implementation slot
var implSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

func NewRateService(ethURL string) (*RateService, error) {
	client, err := ethclient.Dial(ethURL)
	if err != nil {
		return nil, err
	}
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	vault := common.HexToAddress(vaultAddress)
	return &RateService{
		client:    client,
		parsedABI: parsedABI,
		vault:     vault,
	}, nil
}

func NewRateServiceWithCache(ethURL string, c *cache.Cache) (*RateService, error) {
	client, err := ethclient.Dial(ethURL)
	if err != nil {
		return nil, err
	}
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, err
	}
	vault := common.HexToAddress(vaultAddress)
	rs := &RateService{
		client:    client,
		parsedABI: parsedABI,
		vault:     vault,
		cache:     c,
	}
	// Start event log backfill in background
	go rs.EventLogBackfillLast24Hours()
	log.Printf("[EventLogBackfill] Started background event log backfill goroutine")
	return rs, nil
}

func (rs *RateService) FetchAndUpdate() {
	assets, err := callBigInt(rs.client, rs.parsedABI, rs.vault, "totalAssets")
	if err != nil {
		log.Printf("Error calling totalAssets: %v", err)
		return
	}
	supply, err := callBigInt(rs.client, rs.parsedABI, rs.vault, "totalSupply")
	if err != nil {
		log.Printf("Error calling totalSupply: %v", err)
		return
	}
	var rate float64
	if supply.Cmp(big.NewInt(0)) > 0 {
		fAssets := new(big.Float).SetInt(assets)
		fSupply := new(big.Float).SetInt(supply)
		fRate := new(big.Float).Quo(fAssets, fSupply)
		rate, _ = fRate.Float64()
	}
	ts := time.Now().Unix()
	hourTs := ts - (ts % 3600)
	update := models.RateUpdate{
		Timestamp:   hourTs,
		Rate:        rate,
		Assets:      FormatETH(assets),
		TotalSupply: FormatETH(supply),
	}
	if err := rs.cache.SetLatestRate(update); err != nil {
		log.Printf("Error caching latest rate: %v", err)
	}
	if err := rs.cache.AddHistoricalRate(update); err != nil {
		log.Printf("Error caching historical rate: %v", err)
	}
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	if err := rs.cache.CleanupOldRates(cutoff); err != nil {
		log.Printf("Error cleaning up old rates: %v", err)
	}
}

func (rs *RateService) GetLatest() (models.RateUpdate, error) {
	return rs.cache.GetLatestRate()
}

func (rs *RateService) GetHistory(from, to int64) ([]models.RateUpdate, error) {
	return rs.cache.GetHistoricalRates(from, to)
}

func callBigInt(client *ethclient.Client, parsedABI abi.ABI, contract common.Address, method string) (*big.Int, error) {
	data, err := parsedABI.Pack(method)
	if err != nil {
		return nil, err
	}
	callMsg := ethereum.CallMsg{
		To:   &contract,
		Data: data,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, err
	}
	out := new(big.Int)
	parsedABI.UnpackIntoInterface(&out, method, res)
	return out, nil
}

func (rs *RateService) UpdateHourlyHistorical() {
	now := time.Now()
	hourStart := now.Truncate(time.Hour).Unix()
	// Check if this hour's historical value is already in cache
	history, err := rs.cache.GetHistoricalRates(hourStart, hourStart)
	if err == nil && len(history) > 0 {
		// Already cached for this hour
		return
	}
	// Fetch block number for hourStart from Etherscan
	blockNum, err := GetBlockNumberByTimestamp(hourStart)
	if err != nil {
		log.Printf("[HourlyHistorical] Failed to get block number for %d: %v", hourStart, err)
		return
	}
	assetsData, _ := rs.parsedABI.Pack("totalAssets")
	supplyData, _ := rs.parsedABI.Pack("totalSupply")
	assetsHex := "0x" + hex.EncodeToString(assetsData)
	supplyHex := "0x" + hex.EncodeToString(supplyData)
	assetsRes, err := CallContractAtBlock(rs.vault.Hex(), assetsHex, blockNum)
	if err != nil || len(assetsRes) < 3 {
		log.Printf("[HourlyHistorical] Failed to get totalAssets for block=%s: %v", blockNum, err)
		return
	}
	supplyRes, err := CallContractAtBlock(rs.vault.Hex(), supplyHex, blockNum)
	if err != nil || len(supplyRes) < 3 {
		log.Printf("[HourlyHistorical] Failed to get totalSupply for block=%s: %v", blockNum, err)
		return
	}
	assets := new(big.Int)
	supply := new(big.Int)
	assets.SetString(assetsRes[2:], 16)
	supply.SetString(supplyRes[2:], 16)
	var rate float64
	if supply.Cmp(big.NewInt(0)) > 0 {
		fAssets := new(big.Float).SetInt(assets)
		fSupply := new(big.Float).SetInt(supply)
		fRate := new(big.Float).Quo(fAssets, fSupply)
		rate, _ = fRate.Float64()
	}
	update := models.RateUpdate{
		Timestamp:   hourStart,
		Rate:        rate,
		Assets:      FormatETH(assets),
		TotalSupply: FormatETH(supply),
	}
	if err := rs.cache.AddHistoricalRate(update); err != nil {
		log.Printf("[HourlyHistorical] Failed to add historical rate for hour=%d: %v", hourStart, err)
	} else {
		log.Printf("[HourlyHistorical] Added historical rate for hour=%d", hourStart)
	}
}

// Add exported getters for main.go access
func (rs *RateService) ParsedABI() abi.ABI {
	return rs.parsedABI
}

func (rs *RateService) Vault() common.Address {
	return rs.vault
}

func (rs *RateService) Cache() *cache.Cache {
	return rs.cache
}

// GetImplementationAddressAtBlock resolves the implementation address for a proxy at a given block
func GetImplementationAddressAtBlock(client *ethclient.Client, proxy common.Address, blockNum *big.Int) (common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data, err := client.StorageAt(ctx, proxy, implSlot, blockNum)
	if err != nil {
		return common.Address{}, err
	}
	if len(data) < 32 {
		return common.Address{}, fmt.Errorf("invalid storage data")
	}
	return common.BytesToAddress(data[12:]), nil // last 20 bytes
}

// FetchHistoricalValueAtBlockProxy calls the implementation contract at a given block for a proxy
func FetchHistoricalValueAtBlockProxy(client *ethclient.Client, proxy common.Address, abi abi.ABI, method string, blockNum *big.Int) *big.Int {
	implAddr, err := GetImplementationAddressAtBlock(client, proxy, blockNum)
	if err != nil {
		log.Printf("[Proxy] Failed to resolve implementation at block %d: %v", blockNum.Int64(), err)
		return nil
	}
	data, _ := abi.Pack(method)
	res, err := client.CallContract(context.Background(), ethereum.CallMsg{To: &implAddr, Data: data}, blockNum)
	if err != nil {
		log.Printf("[Proxy] eth_call error: %v", err)
		return nil
	}
	var out []interface{}
	err = abi.UnpackIntoInterface(&out, method, res)
	if err != nil {
		log.Printf("[Proxy] unpack error: %v", err)
		return nil
	}
	if len(out) > 0 {
		if v, ok := out[0].(*big.Int); ok {
			return v
		}
		if v, ok := out[0].(big.Int); ok {
			return &v
		}
	}
	return nil
}

// EventLogBackfillLast24Hours reconstructs and caches the last 24 hours of totalSupply/totalAssets using event logs
func (rs *RateService) EventLogBackfillLast24Hours() {
	log.Printf("[EventLogBackfill] Starting EventLogBackfillLast24Hours")
	now := time.Now().Truncate(time.Hour)

	// Get the latest block number
	latestHeader, err := rs.client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Printf("[EventLogBackfill] Failed to get latest block: %v", err)
		return
	}
	latestBlock := latestHeader.Number.Int64()
	fromBlockInt := latestBlock - 1000
	if fromBlockInt < 0 {
		fromBlockInt = 0
	}
	toBlockInt := latestBlock
	log.Printf("[EventLogBackfill] Limiting block range to %d - %d", fromBlockInt, toBlockInt)

	transferSig := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlockInt),
		ToBlock:   big.NewInt(toBlockInt),
		Addresses: []common.Address{rs.vault},
	}
	logs, err := rs.client.FilterLogs(context.Background(), query)
	if err != nil {
		log.Printf("[EventLogBackfill] Failed to fetch logs: %v", err)
		return
	}
	log.Printf("[EventLogBackfill] FilterLogs returned %d logs", len(logs))

	// Pre-fetch all unique block timestamps
	blockTimeMap := make(map[uint64]int64)
	for _, l := range logs {
		if _, ok := blockTimeMap[l.BlockNumber]; !ok {
			block, err := rs.client.BlockByNumber(context.Background(), big.NewInt(int64(l.BlockNumber)))
			if err != nil {
				log.Printf("[EventLogBackfill] Failed to fetch block %d: %v", l.BlockNumber, err)
				continue
			}
			blockTimeMap[l.BlockNumber] = int64(block.Time())
			if len(blockTimeMap)%10 == 0 {
				log.Printf("[EventLogBackfill] Prefetched %d blocks so far...", len(blockTimeMap))
			}
		}
	}
	log.Printf("[EventLogBackfill] Prefetched %d unique block timestamps", len(blockTimeMap))

	// Sort logs by block time ascending
	type logWithTime struct {
		log  types.Log
		time int64
	}
	var logsWithTime []logWithTime
	for _, l := range logs {
		if t, ok := blockTimeMap[l.BlockNumber]; ok {
			logsWithTime = append(logsWithTime, logWithTime{l, t})
		}
	}
	// Sort
	sort.Slice(logsWithTime, func(i, j int) bool { return logsWithTime[i].time < logsWithTime[j].time })

	// Start with the current state (at 'now')
	assets, _ := callBigInt(rs.client, rs.parsedABI, rs.vault, "totalAssets")
	supply, _ := callBigInt(rs.client, rs.parsedABI, rs.vault, "totalSupply")
	log.Printf("[EventLogBackfill] Got current state: Assets=%s, Supply=%s", assets.String(), supply.String())
	currentAssets := new(big.Int).Set(assets)
	currentSupply := new(big.Int).Set(supply)

	isZero := func(addr common.Address) bool {
		return addr == common.Address{}
	}

	hourly := make(map[int64]struct {
		Assets *big.Int
		Supply *big.Int
	})

	// Walk logs backwards (from newest to oldest)
	lastHour := now.Unix()
	for i := len(logsWithTime) - 1; i >= 0; i-- {
		l := logsWithTime[i]
		logHour := l.time - (l.time % 3600)
		// Fill in all hours between lastHour and logHour
		for h := lastHour; h > logHour; h -= 3600 {
			hourly[h] = struct {
				Assets *big.Int
				Supply *big.Int
			}{new(big.Int).Set(currentAssets), new(big.Int).Set(currentSupply)}
			log.Printf("[EventLogBackfill] Writing hour=%d, Assets=%s, Supply=%s", h, currentAssets.String(), currentSupply.String())
		}
		lastHour = logHour
		// Apply event (reverse, since we're going backwards)
		if l.log.Topics[0] == transferSig && len(l.log.Topics) == 3 {
			fromAddr := common.BytesToAddress(l.log.Topics[1].Bytes())
			toAddr := common.BytesToAddress(l.log.Topics[2].Bytes())
			amount := new(big.Int).SetBytes(l.log.Data)
			if isZero(fromAddr) {
				// Mint: reverse by subtracting
				currentSupply.Sub(currentSupply, amount)
				currentAssets.Sub(currentAssets, amount)
			} else if isZero(toAddr) {
				// Burn: reverse by adding
				currentSupply.Add(currentSupply, amount)
				currentAssets.Add(currentAssets, amount)
			}
		}
	}
	// Fill in any remaining hours
	for h := lastHour; h > now.Add(-24*time.Hour).Unix(); h -= 3600 {
		hourly[h] = struct {
			Assets *big.Int
			Supply *big.Int
		}{new(big.Int).Set(currentAssets), new(big.Int).Set(currentSupply)}
		log.Printf("[EventLogBackfill] Writing hour=%d, Assets=%s, Supply=%s", h, currentAssets.String(), currentSupply.String())
	}

	// Insert into cache
	count := 0
	for h, v := range hourly {
		var rate float64
		if v.Supply.Cmp(big.NewInt(0)) > 0 {
			fAssets := new(big.Float).SetInt(v.Assets)
			fSupply := new(big.Float).SetInt(v.Supply)
			fRate := new(big.Float).Quo(fAssets, fSupply)
			rate, _ = fRate.Float64()
		}
		update := models.RateUpdate{
			Timestamp:   h,
			Rate:        rate,
			Assets:      FormatETH(v.Assets),
			TotalSupply: FormatETH(v.Supply),
		}
		err := rs.Cache().AddHistoricalRate(update)
		if err != nil {
			log.Printf("[EventLogBackfill] AddHistoricalRate error for hour=%d: %v", h, err)
		} else {
			count++
		}
	}
	log.Printf("[EventLogBackfill] Inserted %d hourly points", count)
	cutoff := now.Add(-24 * time.Hour).Unix()
	rs.Cache().CleanupOldRates(cutoff)
}
