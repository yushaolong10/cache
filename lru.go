package lru

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var ErrLruNotFoundKey = errors.New("lru key not exist")

type LRUCache struct {
	mutex    sync.Mutex
	maxCount int64                    //max cache key counts
	ttl      int64                    //seconds ttl
	lruList  *list.List               //list
	lruMap   map[string]*list.Element //map

	reqCount int64 //request counts
	hitCount int64 //hit counts
	keyCount int64 //current cache key counts
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
	defer func() {
		cache.checkWithLocked()
		cache.mutex.Unlock()
	}()
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
	}
	return nil
}

func (cache *LRUCache) Get(key string) (interface{}, error) {
	cache.mutex.Lock()
	defer func() {
		cache.checkWithLocked()
		cache.mutex.Unlock()
	}()
	cache.reqCount++
	if ele, ok := cache.lruMap[key]; ok {
		item := ele.Value.(*entry)
		if item.createAt+cache.ttl > time.Now().Unix() { //有效
			cache.hitCount++
			cache.lruList.MoveToBack(ele)
			return item.value, nil
		}
		//expire
		cache.lruList.Remove(ele)
		delete(cache.lruMap, item.key)
		cache.keyCount--
	}
	return nil, ErrLruNotFoundKey
}

func (cache *LRUCache) checkWithLocked() {
	now := time.Now().Unix()
	for cache.lruList.Front() != nil {
		front := cache.lruList.Front()
		e := front.Value.(*entry)
		//key count not greater and key not expired
		if cache.keyCount <= cache.maxCount && now < e.createAt+cache.ttl {
			break
		}
		cache.lruList.Remove(front)
		delete(cache.lruMap, e.key)
		cache.keyCount--
	}
}

func (cache *LRUCache) GetReqCount() int64 {
	return cache.reqCount
}

func (cache *LRUCache) GetHitCount() int64 {
	return cache.hitCount
}

func (cache *LRUCache) GetKeysCount() int64 {
	return cache.keyCount
}
