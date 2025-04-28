package main

import (
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/Zarathos94/puffer/cache"
	"github.com/Zarathos94/puffer/models"
	"github.com/Zarathos94/puffer/routes"
	"github.com/Zarathos94/puffer/utils"
	"github.com/rs/cors"
)

func main() {
	ethURL := os.Getenv("ETH_RPC_URL")
	if ethURL == "" {
		log.Fatal("ETH_RPC_URL environment variable not set")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	c, err := cache.NewCache(redisAddr)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}

	rs, err := utils.NewRateServiceWithCache(ethURL, c)
	if err != nil {
		log.Fatalf("Failed to initialize RateService: %v", err)
	}

	// Start background updater
	go func() {
		var lastCompletedHour int64 = 0
		for {
			now := time.Now()
			// 1. Live update for current value
			rs.FetchAndUpdate() // fetches from contract, caches as "latest"

			// 2. Check if a new hour has completed
			hour := now.Truncate(time.Hour).Unix()
			completedHour := hour - 3600
			if completedHour > lastCompletedHour {
				// Cache the just-completed hour (using Etherscan block at start of completedHour)
				blockNum, err := utils.GetBlockNumberByTimestamp(completedHour)
				if err == nil {
					assetsData, _ := rs.ParsedABI().Pack("totalAssets")
					supplyData, _ := rs.ParsedABI().Pack("totalSupply")
					assetsHex := "0x" + fmt.Sprintf("%x", assetsData)
					supplyHex := "0x" + fmt.Sprintf("%x", supplyData)
					assetsRes, err := utils.CallContractAtBlock(rs.Vault().Hex(), assetsHex, blockNum)
					supplyRes, err2 := utils.CallContractAtBlock(rs.Vault().Hex(), supplyHex, blockNum)
					if err == nil && err2 == nil && len(assetsRes) >= 3 && len(supplyRes) >= 3 {
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
							Timestamp:   completedHour,
							Rate:        rate,
							Assets:      utils.FormatETH(assets),
							TotalSupply: utils.FormatETH(supply),
						}
						rs.Cache().AddHistoricalRate(update)
						// Remove oldest if >24
						cutoff := hour - 24*3600
						rs.Cache().CleanupOldRates(cutoff)
					}
				}
				lastCompletedHour = completedHour
			}
			time.Sleep(15 * time.Second)
		}
	}()

	routes.RegisterRateRoutes(rs)

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(http.DefaultServeMux)

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
