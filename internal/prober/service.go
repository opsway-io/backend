package prober

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	HeartbeatsKey = "probers:heartbeats"
	MetaKey       = "probers:meta"
	HeartbeatTTL  = 60 * time.Second
)

type LocationInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	Status      string `json:"status"` // "online"
}

type Service interface {
	RegisterHeartbeat(ctx context.Context, info LocationInfo) error
	Unregister(ctx context.Context, locationID string) error
	GetActiveLocations(ctx context.Context) ([]LocationInfo, error)
}

type ServiceImpl struct {
	redisClient       *redis.Client
	fallbackLocations []LocationInfo
}

func NewService(redisClient *redis.Client, fallbackLocations []LocationInfo) Service {
	return &ServiceImpl{
		redisClient:       redisClient,
		fallbackLocations: fallbackLocations,
	}
}

func (s *ServiceImpl) RegisterHeartbeat(ctx context.Context, info LocationInfo) error {
	if s.redisClient == nil {
		return nil
	}

	info.Status = "online"
	if info.CountryCode == "" {
		info.CountryCode = ExtractCountryCode(info.ID)
	}
	if info.Name == "" {
		info.Name = info.ID
	}

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	now := float64(time.Now().Unix())

	pipe := s.redisClient.Pipeline()
	pipe.ZAdd(ctx, HeartbeatsKey, redis.Z{
		Score:  now,
		Member: info.ID,
	})
	pipe.HSet(ctx, MetaKey, info.ID, data)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *ServiceImpl) Unregister(ctx context.Context, locationID string) error {
	if s.redisClient == nil {
		return nil
	}

	pipe := s.redisClient.Pipeline()
	pipe.ZRem(ctx, HeartbeatsKey, locationID)
	pipe.HDel(ctx, MetaKey, locationID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *ServiceImpl) GetActiveLocations(ctx context.Context) ([]LocationInfo, error) {
	if s.redisClient == nil {
		return s.fallbackLocations, nil
	}

	cutoff := float64(time.Now().Add(-HeartbeatTTL).Unix())

	// Clean up expired probers
	_ = s.redisClient.ZRemRangeByScore(ctx, HeartbeatsKey, "-inf", strconv.FormatFloat(cutoff, 'f', -1, 64)).Err()

	// Fetch active probers (score >= cutoff)
	locations, err := s.redisClient.ZRevRangeByScore(ctx, HeartbeatsKey, &redis.ZRangeBy{
		Min: strconv.FormatFloat(cutoff, 'f', -1, 64),
		Max: "+inf",
	}).Result()
	if err != nil {
		return s.fallbackLocations, err
	}

	if len(locations) == 0 {
		return s.fallbackLocations, nil
	}

	// Fetch metadata for active probers
	metaMap, err := s.redisClient.HMGet(ctx, MetaKey, locations...).Result()
	if err != nil {
		return s.fallbackLocations, err
	}

	active := make([]LocationInfo, 0, len(locations))
	for i, locID := range locations {
		var info LocationInfo
		if i < len(metaMap) && metaMap[i] != nil {
			if str, ok := metaMap[i].(string); ok {
				_ = json.Unmarshal([]byte(str), &info)
			}
		}
		if info.ID == "" {
			info.ID = locID
			info.Name = locID
			info.CountryCode = ExtractCountryCode(locID)
			info.Status = "online"
		}
		active = append(active, info)
	}

	return active, nil
}

func ExtractCountryCode(locationID string) string {
	loc := strings.ToLower(strings.TrimSpace(locationID))
	if loc == "global" {
		return "random"
	}

	// Check common suffixes e.g. "opsway-dk" -> "dk", "cluster-de" -> "de"
	parts := strings.Split(loc, "-")
	lastPart := parts[len(parts)-1]
	if len(lastPart) == 2 {
		return lastPart
	}

	// Check prefix e.g. "dk-cph" -> "dk"
	firstPart := parts[0]
	if len(firstPart) == 2 {
		return firstPart
	}

	if len(loc) >= 2 {
		return loc[:2]
	}

	return "random"
}
