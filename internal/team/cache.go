package team

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/redis/go-redis/v9"
)

type Cache interface {
	GetTeam(ctx context.Context, teamID uint) (*entities.Team, error)
	SetTeam(ctx context.Context, team *entities.Team, ttl time.Duration) error
	DeleteTeam(ctx context.Context, teamID uint) error
	GetTeamUserCount(ctx context.Context, teamID uint) (int64, error)
	SetTeamUserCount(ctx context.Context, teamID uint, count int64, ttl time.Duration) error
	DeleteTeamUserCount(ctx context.Context, teamID uint) error
}

type CacheImpl struct {
	cli *redis.Client
}

func NewCache(cli *redis.Client) Cache {
	return &CacheImpl{
		cli: cli,
	}
}

func teamCacheKey(teamID uint) string {
	return fmt.Sprintf("team:%d", teamID)
}

func teamUserCountCacheKey(teamID uint) string {
	return fmt.Sprintf("team:%d:user_count", teamID)
}

func (c *CacheImpl) GetTeam(ctx context.Context, teamID uint) (*entities.Team, error) {
	key := teamCacheKey(teamID)
	data, err := c.cli.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var team entities.Team
	if err := json.Unmarshal(data, &team); err != nil {
		return nil, err
	}
	return &team, nil
}

func (c *CacheImpl) SetTeam(ctx context.Context, team *entities.Team, ttl time.Duration) error {
	key := teamCacheKey(team.ID)
	data, err := json.Marshal(team)
	if err != nil {
		return err
	}
	return c.cli.Set(ctx, key, data, ttl).Err()
}

func (c *CacheImpl) DeleteTeam(ctx context.Context, teamID uint) error {
	return c.cli.Del(ctx, teamCacheKey(teamID)).Err()
}

func (c *CacheImpl) GetTeamUserCount(ctx context.Context, teamID uint) (int64, error) {
	key := teamUserCountCacheKey(teamID)
	count, err := c.cli.Get(ctx, key).Int64()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (c *CacheImpl) SetTeamUserCount(ctx context.Context, teamID uint, count int64, ttl time.Duration) error {
	key := teamUserCountCacheKey(teamID)
	return c.cli.Set(ctx, key, count, ttl).Err()
}

func (c *CacheImpl) DeleteTeamUserCount(ctx context.Context, teamID uint) error {
	return c.cli.Del(ctx, teamUserCountCacheKey(teamID)).Err()
}
