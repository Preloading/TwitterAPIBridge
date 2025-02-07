package bridge

import (
	"sync"
	"time"
)

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
	if !found {
		return TwitterUser{}, false
	}
	return user.copy(), true
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

func (u TwitterUser) copy() TwitterUser {
	return TwitterUser{
		Name:                      u.Name,
		ProfileSidebarBorderColor: u.ProfileSidebarBorderColor,
		ProfileBackgroundTile:     u.ProfileBackgroundTile,
		ProfileSidebarFillColor:   u.ProfileSidebarFillColor,
		CreatedAt:                 u.CreatedAt,
		ProfileImageURL:           u.ProfileImageURL,
		Location:                  u.Location,
		ProfileLinkColor:          u.ProfileLinkColor,
		FollowRequestSent:         u.FollowRequestSent,
		URL:                       u.URL,
		FavouritesCount:           u.FavouritesCount,
		ContributorsEnabled:       u.ContributorsEnabled,
		UtcOffset:                 u.UtcOffset,
		ID:                        u.ID,
		ProfileUseBackgroundImage: u.ProfileUseBackgroundImage,
		ProfileTextColor:          u.ProfileTextColor,
		Protected:                 u.Protected,
		FollowersCount:            u.FollowersCount,
		Lang:                      u.Lang,
		Notifications:             u.Notifications,
		TimeZone:                  u.TimeZone,
		Verified:                  u.Verified,
		ProfileBackgroundColor:    u.ProfileBackgroundColor,
		GeoEnabled:                u.GeoEnabled,
		Description:               u.Description,
		FriendsCount:              u.FriendsCount,
		StatusesCount:             u.StatusesCount,
		ProfileBackgroundImageURL: u.ProfileBackgroundImageURL,
		Following:                 u.Following,
		ScreenName:                u.ScreenName,
		ShowAllInlineMedia:        u.ShowAllInlineMedia,
		IsTranslator:              u.IsTranslator,
		ListedCount:               u.ListedCount,
		DefaultProfile:            u.DefaultProfile,
		DefaultProfileImage:       u.DefaultProfileImage,
		Status:                    u.Status,
		ProfileImageURLHttps:      u.ProfileImageURLHttps,
		IDStr:                     u.IDStr,
		ProfileBannerURL:          u.ProfileBannerURL,
		ProfileBannerURLHttps:     u.ProfileBannerURLHttps,
	}
}
