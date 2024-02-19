package farcaster

import (
	"os"
	"sync"
)

var (
	cacheDir = "tmp/framecache"
	once     sync.Once
	Cache    *PfpCache
)

type PfpCache struct {
	m    sync.RWMutex
	pfps map[uint64]string
}

func init() {
	once.Do(func() {
		os.MkdirAll(cacheDir, 0755)
		Cache = &PfpCache{
			pfps: make(map[uint64]string),
		}
	})

}

func (c *PfpCache) GetPfpUrl(fid uint64) string {
	c.m.RLock()
	defer c.m.RUnlock()
	if url, ok := c.pfps[fid]; ok {
		return url
	}
	return ""
}

func (c *PfpCache) SetPfpUrl(fid uint64, url string) {
	c.m.Lock()
	defer c.m.Unlock()
	// bust cache so we don't stay stale
	if len(c.pfps) > 10 {
		c.pfps = make(map[uint64]string)
	}
	c.pfps[fid] = url
}
