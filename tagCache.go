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

	log "github.com/sirupsen/logrus"
)

// TODO:
// 1. Add support for tag going to multiple components
// 2. Add data load method
// 4. Add load tag into DB if not in cache

// TagCache is just a hashmap of tags to tagInfo. Further methods are defined to ease use of the cache
type TagCache struct {
	tags  map[string][]TagInfo
	count int
}

// GetNames gets a []string slice of all tag names in the cache
func (cache *TagCache) GetNames() []string {
	// NOTE: this looks a bit long but it is faster to iterate with
	// i rather than to use append when we already know the size of the slice
	// https://stackoverflow.com/a/27848197
	keys := make([]string, len(cache.tags))
	i := 0
	for k := range cache.tags {
		keys[i] = k
		i++
	}
	return keys
}

// Find gets the tagInfo associated with a tag
func (cache *TagCache) Find(t string) []TagInfo {
	return cache.tags[strings.ToLower(t)]
}

// ContainsTag returns bool if the cache contains the tag
func (cache *TagCache) ContainsTag(t string) bool {
	_, ok := cache.tags[strings.ToLower(t)]
	return ok
}

// ContainsTagInfo returns bool if the cache contains the specific TagInfo
//
func (cache *TagCache) ContainsTagInfo(t TagInfo) bool {
	if cache.ContainsTag(t.Name) {
		for _, tag := range cache.tags[t.Name] {
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
	var err error
	t.Name = strings.ToLower(t.Name)
	if cache.count == 0 || !cache.ContainsTag(t.Name) {
		if err := AddTag(t); err != nil {
			if err == ErrNoComponent {
				return err
			}
			log.Panic(err)
		}
		cache.tags[t.Name], err = QueryTag(t.Name)
		cache.count++
	} else {
		if err := AddTag(t); err != nil {
			if err == ErrNoComponent {
				return err
			}
			log.Panic(err)
		}
		cache.tags[t.Name], err = QueryTag(t.Name)
	}
	if err != nil {
		log.Error("Error fetching tag data from the DB. There may be a discrepancy between the cache and the db")
		log.Panic(err)
	}
	return nil

}

// Drop removes a tag from the cache (probably not necessary for this use case)
// Be aware that this currently removes ALL TagInfo from the cache related to the tag
// FIXME: For consideration - should we remove a specific TagInfo? - Also should we drop from DB?
func (cache *TagCache) Drop(t string) {
	cache.count--
	delete(cache.tags, t)
}

// Load adds all tags in the database to the cache
// This should be called when the cache is first initialized
func (cache *TagCache) Load() {
	cache.tags, cache.count = GetAllTags()
}

// NewTagCache returns a pointer to a tagCache with entries from the database loaded
func NewTagCache() *TagCache {
	var t = new(TagCache)
	t.tags = make(map[string][]TagInfo)
	t.Load() //TODO: Consider adding counter for how long it takes to load the cache? Consider concurrently loading?
	return t
}
