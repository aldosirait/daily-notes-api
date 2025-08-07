package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client     *redis.Client
	defaultTTL time.Duration
}

type CacheConfig struct {
	Host       string
	Port       string
	Password   string
	DB         int
	DefaultTTL time.Duration
}

func NewCacheService(config CacheConfig) *CacheService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v", err)
		return nil
	}

	log.Println("Successfully connected to Redis")

	return &CacheService{
		client:     rdb,
		defaultTTL: config.DefaultTTL,
	}
}

// Set stores a value in cache with TTL
func (cs *CacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if cs == nil || cs.client == nil {
		return fmt.Errorf("cache service not available")
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if ttl == 0 {
		ttl = cs.defaultTTL
	}

	return cs.client.Set(ctx, key, jsonData, ttl).Err()
}

// Get retrieves a value from cache
func (cs *CacheService) Get(ctx context.Context, key string, dest interface{}) error {
	if cs == nil || cs.client == nil {
		return fmt.Errorf("cache service not available")
	}

	val, err := cs.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Delete removes a specific key from cache
func (cs *CacheService) Delete(ctx context.Context, key string) error {
	if cs == nil || cs.client == nil {
		return fmt.Errorf("cache service not available")
	}

	return cs.client.Del(ctx, key).Err()
}

// DeletePattern removes all keys matching a pattern
func (cs *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	if cs == nil || cs.client == nil {
		return fmt.Errorf("cache service not available")
	}

	keys, err := cs.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return cs.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in cache
func (cs *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	if cs == nil || cs.client == nil {
		return false, fmt.Errorf("cache service not available")
	}

	count, err := cs.client.Exists(ctx, key).Result()
	return count > 0, err
}

// GenerateNotesListKey creates a cache key for notes list with filters
func (cs *CacheService) GenerateNotesListKey(userID int, category string, page, limit int) string {
	key := fmt.Sprintf("notes:list:user:%d:page:%d:limit:%d", userID, page, limit)
	if category != "" {
		key += fmt.Sprintf(":category:%s", category)
	}
	return key
}

// GenerateNoteDetailKey creates a cache key for a specific note
func (cs *CacheService) GenerateNoteDetailKey(noteID, userID int) string {
	return fmt.Sprintf("notes:detail:note:%d:user:%d", noteID, userID)
}

// GenerateCategoriesKey creates a cache key for user categories
func (cs *CacheService) GenerateCategoriesKey(userID int) string {
	return fmt.Sprintf("notes:categories:user:%d", userID)
}

// InvalidateUserNotesCache removes all note-related cache for a user
func (cs *CacheService) InvalidateUserNotesCache(ctx context.Context, userID int) error {
	patterns := []string{
		fmt.Sprintf("notes:list:user:%d:*", userID),
		fmt.Sprintf("notes:detail:*:user:%d", userID),
		fmt.Sprintf("notes:categories:user:%d", userID),
	}

	for _, pattern := range patterns {
		if err := cs.DeletePattern(ctx, pattern); err != nil {
			log.Printf("Failed to delete cache pattern %s: %v", pattern, err)
			// Continue with other patterns even if one fails
		}
	}

	return nil
}

// InvalidateNoteCache removes cache for a specific note
func (cs *CacheService) InvalidateNoteCache(ctx context.Context, noteID, userID int) error {
	// Delete specific note detail cache
	detailKey := cs.GenerateNoteDetailKey(noteID, userID)
	if err := cs.Delete(ctx, detailKey); err != nil && err != redis.Nil {
		log.Printf("Failed to delete note detail cache: %v", err)
	}

	// Invalidate all list caches for the user (since note changes affect list results)
	return cs.InvalidateUserNotesCache(ctx, userID)
}

// Health checks if Redis is available
func (cs *CacheService) Health(ctx context.Context) error {
	if cs == nil || cs.client == nil {
		return fmt.Errorf("cache service not available")
	}

	return cs.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (cs *CacheService) Close() error {
	if cs != nil && cs.client != nil {
		return cs.client.Close()
	}
	return nil
}
