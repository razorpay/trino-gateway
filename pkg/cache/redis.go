package cache

import (
	"fmt"
	"time"

	"github.com/go-redis/redis"
)

// RedisConfig holds all required info for initializing redis driver
type RedisConfig struct {
	Host     string
	Port     int32
	Database int32
	Password string
}

// RedisCache holds the handler for the redisclient and auxiliary info
type RedisCache struct {
	redisClient *redis.Client
}

// NewRedisClient inits a RedisCache instance
func NewRedisClient(config *RedisConfig) (*RedisCache, error) {
	var client RedisCache
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       int(config.Database),
	}

	client.redisClient = redis.NewClient(options)

	_, err := client.redisClient.Ping().Result()
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// Set -
func (rc *RedisCache) Set(key string, value string, ttl time.Duration) error {
	err := rc.redisClient.Set(key, value, ttl).Err()
	if err != nil {
		return err
	}

	return nil
}

// Get -
func (rc *RedisCache) Get(key string) (string, error) {
	val, err := rc.redisClient.Get(key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// Delete -
func (rc *RedisCache) Delete(key string) error {
	err := rc.redisClient.Del(key).Err()
	return err
}

//Disconnect ... disconnects from the redis server
func (rc *RedisCache) Disconnect() error {
	err := rc.redisClient.Close()
	if err != nil {
		return err
	}
	return nil
}
