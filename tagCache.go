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

// tagCache is just a hashmap of tags to tagInfo. Further methods are defined to ease use of the cache
type tagCache struct {
	tags  map[string]tagInfo
	count int
}

// Find gets the tagInfo associated with a tag
func (cache *tagCache) Find(t string) tagInfo {
	return cache.tags[t]

}

// Contains returns true/false if the cache contains
func (cache *tagCache) Contains(t string) bool {
	_, ok := cache.tags[t]
	return ok
}

// Add adds a tag + tagInfo to the cache
func (cache *tagCache) Add(t string, tag tagInfo) {
	if cache.count == 0 || !cache.Contains(t) {
		cache.tags[t] = tag
	}
}

// Drop removes a tag from the cache (probably not necessary for this use case)
func (cache *tagCache) Drop(t string) {
	delete(cache.tags, t)
}
