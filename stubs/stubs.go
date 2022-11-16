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
	Code    int               `yaml:"code" json:"code"`
	Data    string            `yaml:"data" json:"data"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

type ServiceMap struct {
	Service map[string]ServiceStub
}

type StubStorage interface {
	InitStorage(cfg *config.StubRouterConfig) error
	GetServiceStubs(host *url.URL) (*ServiceMap, error)
	SaveServiceStub(host *url.URL, path string, data ServiceStub) error
	RemoveServiceStub(host *url.URL, path string) error
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

var redisClient *redis.Client

// InitStorage Inits FS storage
func (s FileStubStorage) InitStorage(cfg *config.StubRouterConfig) error {
	return nil
}

// GetServiceStubs Get all target service stubs data from FS
func (s FileStubStorage) GetServiceStubs(host *url.URL) (*ServiceMap, error) {
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

// SaveServiceStub Save stub data to FS
func (s FileStubStorage) SaveServiceStub(host *url.URL, path string, data ServiceStub) error {
	filename := fmt.Sprintf("%s/%s.yml", s.FsPath, utils.HostToString(host))
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %s", filename)
	}
	defer file.Close()

	var servMap *ServiceMap
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&servMap)
	if err != nil { // File empty, or content not valid yaml, so init empty service map
		servMap = &ServiceMap{Service: make(map[string]ServiceStub)}
	}

	// Clear file
	err = file.Truncate(0)
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	servMap.Service[path] = data
	enc := yaml.NewEncoder(file)
	err = enc.Encode(servMap)
	if err != nil {
		return fmt.Errorf("error writing file: %s", filename)
	}

	return nil
}

func (s FileStubStorage) RemoveServiceStub(host *url.URL, path string) error {
	filename := fmt.Sprintf("%s/%s.yml", s.FsPath, utils.HostToString(host))
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %s", filename)
	}
	defer file.Close()

	var servMap *ServiceMap
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&servMap)
	if err != nil {
		return err
	}

	// Clear file
	err = file.Truncate(0)
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	delete(servMap.Service, path)
	enc := yaml.NewEncoder(file)
	err = enc.Encode(servMap)
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

// GetServiceStubs Get all target service stubs data from Redis
func (s RedisStubStorage) GetServiceStubs(host *url.URL) (*ServiceMap, error) {
	ctx := context.Background()
	val, err := redisClient.HGetAll(ctx, utils.HostToString(host)).Result()
	if err != nil {
		return nil, err
	}

	servMap := &ServiceMap{}
	servMap.Service = make(map[string]ServiceStub)

	for key, value := range val {
		var stub ServiceStub
		err = json.Unmarshal([]byte(value), &stub)
		if err != nil {
			return nil, err
		}

		servMap.Service[key] = stub
	}

	return servMap, nil
}

// SaveServiceStub Save stub data to Redis
func (s RedisStubStorage) SaveServiceStub(host *url.URL, path string, data ServiceStub) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = redisClient.HSet(ctx, utils.HostToString(host), path, val).Err()
	if err != nil {
		return nil
	}

	return nil
}

// RemoveServiceStub Remove service stub from Redis
func (s RedisStubStorage) RemoveServiceStub(host *url.URL, path string) error {
	ctx := context.Background()
	err := redisClient.HDel(ctx, utils.HostToString(host), path).Err()
	if err != nil {
		return err
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

// GetServiceStubs - Get all target service stubs data from store or cache
func (cs *CachedStorage) GetServiceStubs(host *url.URL) (*ServiceMap, error) {
	key := utils.HostToString(host)
	data, found := cs.Cache.Get(key)
	if found {
		hc, ok := data.(*ServiceMap)
		if ok {
			return hc, nil
		}
	}

	s, err := cs.Store.GetServiceStubs(host)
	if err == nil {
		cs.Cache.Set(key, s, cache.DefaultExpiration)
	}

	return s, err
}

// SaveServiceStub Save stub data to store
func (cs *CachedStorage) SaveServiceStub(host *url.URL, path string, data ServiceStub) error {
	// TODO: need clear cache
	return cs.Store.SaveServiceStub(host, path, data)
}

// RemoveServiceStub Remove service stub from cached store
func (cs *CachedStorage) RemoveServiceStub(host *url.URL, path string) error {
	// TODO: need clear cache
	return cs.Store.RemoveServiceStub(host, path)
}
