package bridge

import (
	"sync"
	"time"
)

type User struct {
	ID   string
	Name string
	// Add other fields as needed
}

type Cache struct {
	data  map[string]TwitterUser
	mutex sync.RWMutex
	ttl   time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		data: make(map[string]TwitterUser),
		ttl:  ttl,
	}
}

func (c *Cache) Get(key string) (TwitterUser, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	user, found := c.data[key]
	return user, found
}

func (c *Cache) Set(key string, user TwitterUser) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = user
	go c.expireKeyAfterTTL(key)
}

// maybe not the most effiecent use of memory.
func (c *Cache) SetMultiple(keys []string, user TwitterUser) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for _, key := range keys {
		c.data[key] = user
		go c.expireKeyAfterTTL(key)
	}
}

func (c *Cache) expireKeyAfterTTL(key string) {
	time.Sleep(c.ttl)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
}
