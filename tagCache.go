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

// TODO:
// 1. Add support for tag going to multiple components
// 2. Add data load method
// 4. Add load tag into DB if not in cache

// TagCache is just a hashmap of tags to tagInfo. Further methods are defined to ease use of the cache
type TagCache struct {
	tags  map[string][]TagInfo
	count int
}

// Find gets the tagInfo associated with a tag
func (cache *TagCache) Find(t string) []TagInfo {
	return cache.tags[t]
}

// Contains returns true/false if the cache contains the tag
func (cache *TagCache) Contains(t string) bool {
	_, ok := cache.tags[t]
	return ok
}

// Add adds a tag + TagInfo to the cache. If the tag is already in the cache, it adds
// to the TagInfo array
func (cache *TagCache) Add(t string, tag TagInfo) {
	if cache.count == 0 || !cache.Contains(t) {
		cache.tags[t] = []TagInfo{tag}
		cache.count++
		AddTag(tag)
	} else {
		cache.tags[t] = append(cache.tags[t], tag)
		AddTag(tag)
	}
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
func (cache *TagCache) Load() error {
	// TODO: FINISH HIM
	return nil
}

// NewTagCache returns a pointer to a tagCache with entries from the database loaded
func NewTagCache() *TagCache {
	var t = new(TagCache)
	t.Load()
	return t
}
