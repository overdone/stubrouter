package stubs

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/overdone/stubrouter/config"
	"github.com/overdone/stubrouter/utils"
	"github.com/patrickmn/go-cache"
	"gopkg.in/yaml.v3"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type ServiceStub struct {
	Code    int               `yaml:"code"`
	Data    string            `yaml:"data"`
	Headers map[string]string `yaml:"headers"`
}

type ServiceMap struct {
	Service map[string]ServiceStub
}

type StubStorage interface {
	InitStorage(cfg *config.StubRouterConfig) error
	GetServiceMap(host *url.URL) (*ServiceMap, error)
	SetServiceMap(host *url.URL, data ServiceMap) error
}

type FileStubStorage struct {
	FsPath string
}

type RedisStubStorage struct {
	ConnString string
}

type CachedStorage struct {
	Store StubStorage
	Cache *cache.Cache
}

var ctx context.Context
var redisClient *redis.Client

// InitStorage Inits FS storage
func (s FileStubStorage) InitStorage(cfg *config.StubRouterConfig) error {
	return nil
}

// GetServiceMap Get host data from FS
func (s FileStubStorage) GetServiceMap(host *url.URL) (*ServiceMap, error) {
	filename := fmt.Sprintf("%s/%s.yml", s.FsPath, utils.HostToString(host))
	file, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servMap *ServiceMap
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&servMap)
	if err != nil {
		return nil, err
	}

	return servMap, nil
}

// SetServiceMap Set host data tp FS
func (s FileStubStorage) SetServiceMap(host *url.URL, data ServiceMap) error {
	// TODO: make it thread safe
	filename := fmt.Sprintf("%s/%s.yml", s.FsPath, utils.HostToString(host))
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %s", filename)
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)
	err = enc.Encode(data)
	if err != nil {
		return fmt.Errorf("error writing file: %s", filename)
	}

	return nil
}

// InitStorage Inits Redis DB storage
func (s RedisStubStorage) InitStorage(cfg *config.StubRouterConfig) error {
	opts, err := redis.ParseURL(cfg.Stubs.Storage.Path)
	if err != nil {
		return fmt.Errorf("invalid redis path")
	}

	redisClient = redis.NewClient(opts)
	return nil
}

// GetServiceMap Get host data from Redis
func (s RedisStubStorage) GetServiceMap(host *url.URL) (*ServiceMap, error) {
	val, err := redisClient.Get(ctx, utils.HostToString(host)).Result()
	if err != nil {
		return nil, err
	}

	var servMap *ServiceMap
	err = json.Unmarshal([]byte(val), servMap)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// SetServiceMap Set host data to Redis
func (s RedisStubStorage) SetServiceMap(host *url.URL, data ServiceMap) error {
	err := redisClient.Set(ctx, utils.HostToString(host), data, 0).Err()
	if err != nil {
		return fmt.Errorf("error writing to Redis DB")
	}

	return nil
}

// InitStorage - Inits cached cache
func (cs *CachedStorage) InitStorage(cfg *config.StubRouterConfig) error {
	expirationInterval, err := time.ParseDuration(cfg.Stubs.Storage.Cache.ExpirationInterval)
	cleanupInterval, err := time.ParseDuration(cfg.Stubs.Storage.Cache.CleanupInterval)
	if err != nil {
		log.Fatal(">>> Config error. Invalid stub config param")
	}

	cs.Cache = cache.New(expirationInterval, cleanupInterval)
	return cs.Store.InitStorage(cfg)
}

// GetServiceMap - Get host data from store or cache
func (cs *CachedStorage) GetServiceMap(host *url.URL) (*ServiceMap, error) {
	key := utils.HostToString(host)
	data, found := cs.Cache.Get(key)
	if found {
		hc, ok := data.(*ServiceMap)
		if ok {
			return hc, nil
		}
	}

	s, err := cs.Store.GetServiceMap(host)
	if err == nil {
		cs.Cache.Set(key, s, cache.DefaultExpiration)
	}

	return s, err
}

// SetServiceMap - Set host data to store
func (cs *CachedStorage) SetServiceMap(host *url.URL, data ServiceMap) error {
	return cs.SetServiceMap(host, data)
}
