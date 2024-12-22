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
	data  map[string]User
	mutex sync.RWMutex
	ttl   time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		data: make(map[string]User),
		ttl:  ttl,
	}
}

func (c *Cache) Get(key string) (User, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	user, found := c.data[key]
	return user, found
}

func (c *Cache) Set(key string, user User) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = user
	go c.expireKeyAfterTTL(key)
}

func (c *Cache) expireKeyAfterTTL(key string) {
	time.Sleep(c.ttl)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
}
