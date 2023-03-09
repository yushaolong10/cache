package cache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var ErrKeyNotFound = errors.New("key not found")

type LRUCache struct {
	mutex    sync.Mutex
	maxCount int64                    //max cache key counts
	ttl      int64                    //seconds ttl
	lruList  *list.List               //list
	lruMap   map[string]*list.Element //map
	keyCount int64                    //current cache key counts

	totalReqTimes int64 //total request times
	totalHitTimes int64 //total hit times
}

type entry struct {
	key      string
	value    interface{}
	createAt int64 //create unix timestamp
}

func NewLRUCache(maxCount int, ttl int) *LRUCache {
	cache := &LRUCache{
		maxCount: int64(maxCount),
		ttl:      int64(ttl),
		lruList:  list.New(),
		lruMap:   make(map[string]*list.Element),
	}
	return cache
}

func (cache *LRUCache) Update(key string, value interface{}) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if ele, ok := cache.lruMap[key]; ok { //exist
		item := ele.Value.(*entry)
		item.value = value
		item.createAt = time.Now().Unix()
		cache.lruList.MoveToBack(ele)
	} else { //new
		item := &entry{
			key:      key,
			value:    value,
			createAt: time.Now().Unix(),
		}
		cache.lruMap[key] = cache.lruList.PushBack(item)
		cache.keyCount++
		cache.checkWithLocked()
	}
	return nil
}

func (cache *LRUCache) Get(key string) (interface{}, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.totalReqTimes++
	if ele, ok := cache.lruMap[key]; ok {
		item := ele.Value.(*entry)
		if item.createAt+cache.ttl > time.Now().Unix() { //有效
			cache.totalHitTimes++
			cache.lruList.MoveToBack(ele)
			return item.value, nil
		}
		//expire
		cache.lruList.Remove(ele)
		delete(cache.lruMap, key)
		cache.keyCount--
	}
	return nil, ErrKeyNotFound
}

func (cache *LRUCache) Delete(key string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.totalReqTimes++
	if ele, ok := cache.lruMap[key]; ok {
		cache.totalHitTimes++
		cache.lruList.Remove(ele)
		delete(cache.lruMap, key)
		cache.keyCount--
		return nil
	}
	return ErrKeyNotFound
}

func (cache *LRUCache) checkWithLocked() {
	for cache.keyCount > cache.maxCount && cache.lruList.Front() != nil {
		front := cache.lruList.Front()
		item := front.Value.(*entry)
		cache.lruList.Remove(front)
		delete(cache.lruMap, item.key)
		cache.keyCount--
	}
}

func (cache *LRUCache) GetKeyCount() int64 {
	return cache.keyCount
}

func (cache *LRUCache) GetTotalReqTimes() int64 {
	return cache.totalReqTimes
}

func (cache *LRUCache) GetTotalHitTimes() int64 {
	return cache.totalHitTimes
}
