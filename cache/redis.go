package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/Zarathos94/puffer/models"
	"github.com/redis/go-redis/v9"
)

const (
	RedisRateKey    = "latest_rate"
	RedisHistoryKey = "rate_history"
)

type Cache struct {
	client *redis.Client
}

func NewCache(redisAddr string) (*Cache, error) {
	if redisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR must be set")
	}
	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return &Cache{client: client}, nil
}

func (c *Cache) SetLatestRate(rate models.RateUpdate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	b, err := json.Marshal(rate)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, RedisRateKey, b, 0).Err()
}

func (c *Cache) GetLatestRate() (models.RateUpdate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := c.client.Get(ctx, RedisRateKey).Result()
	if err != nil {
		return models.RateUpdate{}, err
	}
	var rate models.RateUpdate
	if err := json.Unmarshal([]byte(val), &rate); err != nil {
		return models.RateUpdate{}, err
	}
	return rate, nil
}

func (c *Cache) AddHistoricalRate(rate models.RateUpdate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Round timestamp to the start of the hour
	hourTs := rate.Timestamp - (rate.Timestamp % 3600)
	rate.Timestamp = hourTs
	b, err := json.Marshal(rate)
	if err != nil {
		return err
	}
	// Remove any existing rate for this hour
	results, err := c.client.ZRangeByScore(ctx, RedisHistoryKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(hourTs, 10),
		Max: strconv.FormatInt(hourTs, 10),
	}).Result()
	if err == nil && len(results) > 0 {
		for _, v := range results {
			_ = c.client.ZRem(ctx, RedisHistoryKey, v).Err()
		}
	}
	return c.client.ZAdd(ctx, RedisHistoryKey, redis.Z{
		Score:  float64(hourTs),
		Member: b,
	}).Err()
}

func (c *Cache) GetHistoricalRates(from, to int64) ([]models.RateUpdate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	results, err := c.client.ZRangeByScore(ctx, RedisHistoryKey, &redis.ZRangeBy{
		Min: strconv.FormatInt(from, 10),
		Max: strconv.FormatInt(to, 10),
	}).Result()
	if err != nil {
		return nil, err
	}
	ratesMap := make(map[int64]models.RateUpdate)
	for _, v := range results {
		var rate models.RateUpdate
		if err := json.Unmarshal([]byte(v), &rate); err == nil {
			// Only keep one rate per hour (the last one found)
			ratesMap[rate.Timestamp] = rate
		}
	}
	// Convert map to slice and sort by timestamp ascending
	rates := make([]models.RateUpdate, 0, len(ratesMap))
	for _, r := range ratesMap {
		rates = append(rates, r)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i].Timestamp < rates[j].Timestamp })
	log.Printf("[GetHistoricalRates] Returning %d hourly points from %d to %d", len(rates), from, to)
	return rates, nil
}

func (c *Cache) CleanupOldRates(cutoff int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.client.ZRemRangeByScore(ctx, RedisHistoryKey, "-inf", fmt.Sprintf("(%d", cutoff)).Err()
}

func (c *Cache) GetLastHistoricalTimestamp() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res, err := c.client.ZRevRangeWithScores(ctx, RedisHistoryKey, 0, 0).Result()
	if err != nil || len(res) == 0 {
		return 0, err
	}
	return int64(res[0].Score), nil
}
