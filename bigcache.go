package bigcache

import (
	"fmt"
	"log"
	"sync"

	"github.com/ijibu/bigcache/queue"
)

const (
	minimumEntriesInShard = 10 // Minimum number of entries in single shard，单个分区里面最小的缓存实体容量
)

// BigCache is fast, concurrent, evicting cache created to keep big number of entries without impact on performance.
// It keeps entries on heap but omits GC for them. To achieve that operations on bytes arrays take place,
// therefore entries (de)serialization in front of the cache will be needed in most use cases.
type BigCache struct {
	shards       []*cacheShard //分区链
	lifeWindow   uint64        //生命周期
	clock        clock
	hash         Hasher //计算hash的
	config       Config //配置信息段
	shardMask    uint64
	maxShardSize uint32 //最大分区数量
}

//一个缓存分区块
type cacheShard struct {
	hashmap     map[uint64]uint32 //哈希表
	entries     queue.BytesQueue  //缓存实体队列FIFO
	lock        sync.RWMutex      //锁
	entryBuffer []byte            //实体缓冲，每次往缓存中写入数据时，都会用这个缓冲，避免每次去分配内存。
}

// NewBigCache initialize new instance of BigCache
func NewBigCache(config Config) (*BigCache, error) {
	return newBigCache(config, &systemClock{})
}

func newBigCache(config Config, clock clock) (*BigCache, error) {

	if !isPowerOfTwo(config.Shards) {
		return nil, fmt.Errorf("Shards number must be power of two")
	}

	if config.Hasher == nil {
		config.Hasher = newDefaultHasher()
	}

	//每个分区的内存大小
	maxShardSize := 0
	if config.HardMaxCacheSize > 0 {
		maxShardSize = convertMBToBytes(config.HardMaxCacheSize) / config.Shards
	}

	cache := &BigCache{
		shards:       make([]*cacheShard, config.Shards),
		lifeWindow:   uint64(config.LifeWindow.Seconds()),
		clock:        clock,
		hash:         config.Hasher,
		config:       config,
		shardMask:    uint64(config.Shards - 1),
		maxShardSize: uint32(maxShardSize),
	}

	//初始缓存分区块的大小，一个分区块可以放多少个缓存项
	initShardSize := max(config.MaxEntriesInWindow/config.Shards, minimumEntriesInShard)
	for i := 0; i < config.Shards; i++ {
		cache.shards[i] = &cacheShard{
			hashmap:     make(map[uint64]uint32, initShardSize), //性能考虑，事先分配好map的大小。
			entries:     *queue.NewBytesQueue(initShardSize*config.MaxEntrySize, maxShardSize, config.Verbose),
			entryBuffer: make([]byte, config.MaxEntrySize+headersSizeInBytes),
		}
	}

	return cache, nil
}

func isPowerOfTwo(number int) bool {
	return (number & (number - 1)) == 0
}

// Get reads entry for the key
func (c *BigCache) Get(key string) ([]byte, error) {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	shard.lock.RLock()
	defer shard.lock.RUnlock()

	itemIndex := shard.hashmap[hashedKey]

	if itemIndex == 0 {
		return nil, notFound(key)
	}

	wrappedEntry, err := shard.entries.Get(int(itemIndex))
	if err != nil {
		return nil, err
	}
	if entryKey := readKeyFromEntry(wrappedEntry); key != entryKey {
		if c.config.Verbose {
			log.Printf("Collision detected. Both %q and %q have the same hash %x", key, entryKey, hashedKey)
		}
		return nil, notFound(key)
	}
	return readEntry(wrappedEntry), nil
}

// Set saves entry under the key
func (c *BigCache) Set(key string, entry []byte) error {
	hashedKey := c.hash.Sum64(key)
	shard := c.getShard(hashedKey)
	shard.lock.Lock()
	defer shard.lock.Unlock()

	currentTimestamp := uint64(c.clock.epoch())

	if previousIndex := shard.hashmap[hashedKey]; previousIndex != 0 {
		if previousEntry, err := shard.entries.Get(int(previousIndex)); err == nil {
			resetKeyFromEntry(previousEntry)
		}
	}

	if oldestEntry, err := shard.entries.Peek(); err == nil {
		c.onEvict(oldestEntry, currentTimestamp, shard.removeOldestEntry)
	}

	w := wrapEntry(currentTimestamp, hashedKey, key, entry, &shard.entryBuffer)

	//首先往队列里面放，如果放不进去再移除最老的实体。如此循环，直到成功放进去或者移除实体失败返回false为止。
	for {
		if index, err := shard.entries.Push(w); err == nil {
			shard.hashmap[hashedKey] = uint32(index)
			return nil
		} else if shard.removeOldestEntry() != nil {
			return fmt.Errorf("Entry is bigger than max shard size.")
		}
	}
}

//淘汰最老的过期的缓存
func (c *BigCache) onEvict(oldestEntry []byte, currentTimestamp uint64, evict func() error) {
	oldestTimestamp := readTimestampFromEntry(oldestEntry)
	//如果过期了，则进行淘汰
	if currentTimestamp-oldestTimestamp > c.lifeWindow {
		evict()
	}
}

//删除最老的一个缓存，不管过没过期
func (s *cacheShard) removeOldestEntry() error {
	oldest, err := s.entries.Pop()
	if err == nil {
		hash := readHashFromEntry(oldest)
		//从hashmap中删除一个元素
		delete(s.hashmap, hash)
		return nil
	}
	return err
}

//根据hash值获取分区
func (c *BigCache) getShard(hashedKey uint64) (shard *cacheShard) {
	return c.shards[hashedKey&c.shardMask]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func convertMBToBytes(value int) int {
	return value * 1024 * 1024
}
