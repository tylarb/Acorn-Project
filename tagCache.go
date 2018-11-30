/*
Local cache for tags to enable faster processing and more natural interaction with bot.

Should load tags into cache at bot start and handle updating tags into DB and cache when they are added

Simple usage example:

if cache.Contains(tag) {
	tagInfo = cache.Find(tag)
	// handle printing tagInfo  -> can also handle "this tage is already marked a component"
} else {
	tagInfo = nil
	// don't print tagInfo
}

*/

package main

import (
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// TagCache is just a hashmap of tags to tagInfo. Further methods are defined to ease use of the cache
type TagCache struct {
	sync.Mutex
	Tags  map[string][]TagInfo
	Count int
}

// TagInfo is the response structure when a tag query is made
type TagInfo struct {
	Anchor        string
	Name          string
	PlaybookURL   string
	ComponentChan string
	SupportChan   string
}

// GetNames gets a []string slice of all tag names in the cache
func (cache *TagCache) GetNames() []string {
	cache.Lock()
	defer cache.Unlock()
	return cache.getNames()
}
func (cache *TagCache) getNames() []string {
	// NOTE: this looks a bit long but it is faster to iterate with
	// i rather than to use append when we already know the size of the slice
	// https://stackoverflow.com/a/27848197
	keys := make([]string, cache.Count)
	i := 0
	for k := range cache.Tags {
		keys[i] = k
		i++
	}
	return keys

}

// Find gets the tagInfo associated with a tag
func (cache *TagCache) Find(t string) []TagInfo {
	cache.Lock()
	defer cache.Unlock()
	return cache.find(t)
}

func (cache *TagCache) find(t string) []TagInfo {
	return cache.Tags[strings.ToLower(t)]

}

// ContainsTag returns bool if the cache contains the tag
func (cache *TagCache) ContainsTag(t string) bool {
	cache.Lock()
	defer cache.Unlock()
	return cache.containsTag(t)
}

func (cache *TagCache) containsTag(t string) bool {
	_, ok := cache.Tags[strings.ToLower(t)]
	return ok
}

// ContainsTagInfo returns bool if the cache contains the specific TagInfo
func (cache *TagCache) ContainsTagInfo(t TagInfo) bool {
	cache.Lock()
	defer cache.Unlock()
	return cache.containsTagInfo(t)
}

func (cache *TagCache) containsTagInfo(t TagInfo) bool {
	t.Name = strings.ToLower(t.Name)
	if cache.containsTag(t.Name) {
		for _, tag := range cache.Tags[t.Name] {
			if tag.ComponentChan == t.ComponentChan {
				return true
			}
		}
	}
	return false
}

// Add adds a tag + TagInfo to the cache. If the tag is already in the cache, it adds
// to the TagInfo array. Handles lowering strings as well
func (cache *TagCache) Add(t TagInfo) error {
	cache.Lock()
	defer cache.Unlock()
	return cache.add(t)

}

func (cache *TagCache) add(t TagInfo) error {
	var err error
	t.Name = strings.ToLower(t.Name)
	if cache.Count == 0 || !cache.containsTag(t.Name) {
		if err := AddTag(t); err != nil {
			if err == ErrNoComponent {
				return err
			}
			log.Panic(err) // we don't want a discrepancy between cache and there's some critical issue here
			// TODO : error handling wehre we alert the maintainer that there's an issue
		}
		cache.Tags[t.Name], err = QueryTag(t.Name)
		cache.Count++
	} else {
		if err := AddTag(t); err != nil {
			if err == ErrNoComponent {
				return err
			}
			log.Panic(err)
		}
		cache.Tags[t.Name], err = QueryTag(t.Name)
	}
	if err != nil {
		log.Error("Error fetching tag data from the DB. There may be a discrepancy between the cache and the db")
		log.Panic(err) // TODO alert bot maintainer
	}
	return nil
}

// Drop removes a tag from the cache (probably not necessary for this use case)
// Be aware that this currently removes ALL TagInfo from the cache related to the tag
// FIXME: For consideration - should we remove a specific TagInfo? - Also should we drop from DB?
func (cache *TagCache) Drop(t string) {
	cache.Lock()
	defer cache.Unlock()
	cache.drop(t)
}

func (cache *TagCache) drop(t string) {
	cache.Count--
	delete(cache.Tags, t)
}

// Load adds all tags in the database to the cache  // TODO - govern concurrent access here?
// This should be called when the cache is first initialized
func (cache *TagCache) Load() {
	cache.Lock()
	defer cache.Unlock()
	cache.load()
}

func (cache *TagCache) load() {
	cache.Tags, cache.Count = GetAllTags()
}

// NewTagCache returns a pointer to a tagCache with entries from the database loaded
func NewTagCache() *TagCache {
	var t = new(TagCache)
	t.Tags = make(map[string][]TagInfo)
	t.Load() //TODO: Consider adding counter for how long it takes to load the cache? Consider concurrently loading?
	return t
}
