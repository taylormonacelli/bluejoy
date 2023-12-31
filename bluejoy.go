package bluejoy

import (
	"encoding/gob"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	gocache "github.com/patrickmn/go-cache"
	mymazda "github.com/taylormonacelli/forestfish/mymazda"
	"github.com/taylormonacelli/somespider"
)

var relCachePath string

func init() {
	relCachePath = filepath.Join("bluejoy", "data.db")
}

func Main() int {
	dataPath, _ := somespider.GenPath(relCachePath)
	slog.Debug("cache", "path", dataPath)

	cache1 := gocache.New(3*time.Minute, 4*time.Minute)
	slog.Debug("cache", "exists", mymazda.FileExists(dataPath))

	slog.Debug("cache", "action", "deleting cache file")

	// ensure we're starting clean for testing/iterating:
	os.Remove(dataPath)
	slog.Debug("cache", "exists", mymazda.FileExists(dataPath))

	// generate domain item we'd like to cache and cache it:
	cacheItem := PushbulletHTTReply{
		Pushes: []Push{
			{URL: "https://news.ycombinator.com/"},
			{URL: "https://go.dev/blog/gob"},
		},
	}
	cache1.Set("foo", cacheItem, gocache.DefaultExpiration)
	slog.Debug("check in memory cache items", "count", cache1.ItemCount())

	// prepare to persist cache to disk:
	cacheSnapshot := cache1.Items()

	gob.Register(PushbulletHTTReply{})

	// serialize using gob:
	file, _ := os.Create(dataPath)
	encoder := gob.NewEncoder(file)
	err := encoder.Encode(cacheSnapshot)
	if err != nil {
		slog.Error("encode", "error", err.Error())
	}
	defer file.Close()
	slog.Debug("checking existance of file cache", "exists", mymazda.FileExists(dataPath))

	// unmarshal cache from file:
	file2, err := os.Open(dataPath)
	if err != nil {
		slog.Debug("file access", "error", err.Error())
		return 1
	}
	defer file2.Close()

	gobDecoder := gob.NewDecoder(file2)

	var cacheMap map[string]gocache.Item
	if err := gobDecoder.Decode(&cacheMap); err != nil {
		slog.Debug("decode", "error", err.Error())
		return 1
	}

	cache2 := gocache.NewFrom(1*time.Minute, 2*time.Minute, cacheMap)
	reply, future, found := cache2.GetWithExpiration("foo")

	expires := time.Until(future).Truncate(time.Second)
	e := reply.(PushbulletHTTReply)
	slog.Debug("newCache", "found", found, "expires", expires, "now", time.Now(), "future", future, "reply", e.Pushes[0].URL)

	return 0
}
