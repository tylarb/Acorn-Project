/*
All RTM messages are sent here in order to be parsed. All commands are of one
of three types:

1. Message at bot > command function
2. Bookmark query of format <bookmark>? [one word long]
3. any other message, which is parsed for karma, or shame

Because all messages are either logs or printed to slack, a slack client is
defined at the main package in order to reduce passing the slack client around



Released under MIT license, copyright 2018 Tyler Ramer

*/

package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	lv "github.com/texttheater/golang-levenshtein/levenshtein"
)

type response struct {
	message     string
	user        string
	channel     string
	isEphemeral bool
	isIM        bool
	threadTS    string
}

// LV tuning parameters
var (
	matchDistPercent = .85
	minWordLength    = 4
)

// regex definitions

var (
	regHelp   = regexp.MustCompile(`^(?i)help[\?]?$`)
	regTags   = regexp.MustCompile(`^(?i)tag[s]?[\:]?$`)
	regAdd    = regexp.MustCompile(`(?i)add$`)
	regAnchor = regexp.MustCompile(`(?i)anchor$`)
	regSet    = regexp.MustCompile(`(?i)set$`)
)

// parses all messagess from slack for special commands or karma events
func parse(ev *slack.MessageEvent) (err error) {
	var atBot = fmt.Sprintf("<@%s>", botID)
	if ev.User == "USLACKBOT" {
		log.Debug("Slackbot sent a message which is ignored")
		return nil
	}
	words := strings.Fields(ev.Text)
	switch {
	case words[0] == atBot:
		log.WithField("Message", ev.Text).Debug("Instuction for bot")
		err = handleCommand(ev, words)
	default:
		log.WithField("Message", ev.Text).Debug("Handling individual words")
		err = handleWord(ev, words)
	}

	return nil
}

// regex match and take appropriate action on words in a sentance. This only gets executed if
// the message is not deemed some other "type" of interation - like a command to the bot
func handleWord(ev *slack.MessageEvent, words []string) (err error) {
	switch {
	case regHelp.MatchString(words[0]):
		log.Debug("handling a help message")
		handleHelp(ev, words)
	case regTags.MatchString(words[0]):
		log.Debug("Handling a tag message")
		if len(words) > 1 {
			handleKeywords(ev, words)
		} else {
			postHelp(ev, baseHelp)
		}
	case regAnchor.MatchString(words[0]):
		log.Debug("handling an anchor message")
		if len(words) > 1 {
			handleAnchor(ev, words)
		} else {
			postHelp(ev, baseHelp)
		}
		// TODO add SC integration and allow cases to be passed
		/*		case askCase.MatchString(words[0]):
				if len(words) > 1 {
					handleCase(ev, case)
				} else { postHelp(ev)} */
	}

	return nil

}

// Handles help requests  TODO: Add help for adding to database, etc

func handleHelp(ev *slack.MessageEvent, words []string) error {
	switch {
	case len(words) == 1:
		postHelp(ev, baseHelp)
	case len(words) > 1 && regTags.MatchString(words[1]):
		postHelp(ev, tagsHelp)
	case len(words) > 1 && regAdd.MatchString(words[1]):
		postHelp(ev, addHelp)
	case len(words) > 1 && regAnchor.MatchString(words[1]):
		postHelp(ev, anchorHelp)
	default:
		postHelp(ev, baseHelp)
	}

	return nil
}

// handlesKeywords passed via the "tag" option
func handleKeywords(ev *slack.MessageEvent, words []string) error {
	var (
		responses  []string
		foundMatch bool
	)
	r := response{user: ev.User, channel: ev.Channel, isEphemeral: false, isIM: false}
	r.setResponseContext(ev)

	responses, foundMatch = tagMatch(words)
	if foundMatch {
		r.message = strings.Join(responses[:], "\n")
	} else {
		r.message = noRelevantTag
	}
	slackPrint(r)
	return nil

}

func tagMatch(words []string) (responses []string, match bool) {
	tagsCache := cache.GetNames()
	incomingTags := make(chan TagInfo)

	words = words[1:]
	complete := make(chan bool)
	defer close(complete)
	c1 := 0
	for _, word := range words {
		go lvSearch(word, tagsCache, incomingTags, complete)
		c1++

	}

	c2 := 0
	for i := 0; i < len(words)-1; i++ {
		word := strings.Join(words[i:i+2], " ")
		go lvSearch(word, tagsCache, incomingTags, complete)
		c2++
	}

	c3 := 0
	for i := 0; i < len(words)-2; i++ {
		word := strings.Join(words[i:i+3], " ")
		go lvSearch(word, tagsCache, incomingTags, complete)
		c3++
	}

	c := c1 + c2 + c3
	go func(c int, complete chan bool, incomingTags chan TagInfo) {
		counter := 0
		for range complete {
			counter++
			if counter == c {
				close(incomingTags)
				break
			}
		}
	}(c, complete, incomingTags)
	for tag := range incomingTags {
		match = true
		responses = append(responses, tagFmt(tag))
	}
	return responses, match
}

func lvSearch(word string, tagsCache []string, found chan TagInfo, complete chan bool) {
	if len(word) < minWordLength {
		if cache.ContainsTag(word) {
			log.WithField("tag", word).Debug("found exact match in cache")
			for _, tag := range cache.Find(word) {
				found <- tag
			}
		}
	} else {
		if cache.ContainsTag(word) {
			log.WithField("tag", word).Debug("found exact match in cache")
			for _, tag := range cache.Find(word) {
				found <- tag

			}
		} else {
			for _, t := range tagsCache {
				if len(t) < minWordLength {
					continue
				}
				dist := lv.RatioForStrings([]rune(word), []rune(t), lv.DefaultOptions)
				log.WithFields(log.Fields{"s1": t, "s2": word, "dist": dist}).Debug("Levenshtein ratio")
				if dist == 1 {
					log.WithField("tag", t).Error("Using lv dist for an exact match - this should not have occured")
					break // Due to exact match, the response should be populated.
				} else if dist >= matchDistPercent {
					log.WithField("tag", t).Debug("Found fuzzy match")
					for _, tag := range cache.Find(t) {
						found <- tag
					}
				}

			}
		}
	}
	complete <- true
}

func handleAnchor(ev *slack.MessageEvent, words []string) error {
	r := response{user: ev.User, channel: ev.Channel}
	r.setResponseContext(ev)

	word := words[1]
	component, err := GetAnchor(chanTrim(word))
	if err != nil {
		if err == ErrNoComponent {
			r.message = noComponentInDB
			slackPrint(r)
		} else if err == ErrNoChannel {
			r.message = noChannelInSlack
			slackPrint(r)
		} else {
			log.Panic(err)
		}
	} else {
		r.message = componentFmt(component)
	}
	slackPrint(r)
	return nil
}

// TODO: add Servicecloud integration and scan cases for details
/*func handleCase(ev *slack.MessageEvent, words []string) error {
	return nil
}*/

// Commands directed at the bot
func handleCommand(ev *slack.MessageEvent, words []string) error {
	r := response{user: ev.User, channel: ev.Channel, isEphemeral: true}
	switch {
	case regTags.MatchString(words[1]):
		if len(words) < 4 {
			postHelp(ev, tagsHelp)
			return nil
		} // TODO: clean this up
		tag := TagInfo{ComponentChan: chanTrim(words[2])}
		count := 0
		tagList := tagCleanup(ev.Text)
		for _, word := range tagList {
			tag.Name = word
			if !cache.ContainsTagInfo(tag) {
				if err := cache.Add(tag); err != nil {
					if err == ErrNoComponent {
						r.message = noComponentInDB
						slackPrint(r)
						break
					} else if err == ErrNoChannel {
						r.message = noChannelInSlack
						slackPrint(r)
						break
					} else if err == ErrTagTooLong {
						r.message = fmt.Sprintf(tagTooLong, tag.Name)
						slackPrint(r)

					}
				}
				count++
			} else {
				r.message = fmt.Sprintf(alreadyAdded, tag.Name)
				slackPrint(r)
			}
		}
		if count != 0 {
			r.message = fmt.Sprintf("Added %d tags to the component %s", count, words[2])
			slackPrint(r)
		}
	case regAdd.MatchString(words[1]):
		switch {
		case len(words) < 5:
			postHelp(ev, addHelp)
		case words[2] == "component":
			postHelp(ev, addHelp) // TODO finish building matrix/DB to add component
		case words[4] == "as":
			postHelp(ev, addHelp) // TODO finish building matrix/DB to add anchor
		}
	case regHelp.MatchString(words[1]):
		handleHelp(ev, words[1:])
	case regSet.MatchString(words[1]):
		if len(words) < 5 || !regAnchor.MatchString(words[3]) {
			postHelp(ev, anchorHelp)
			return nil
		}
		if !validateAnchorName(usrTrim(words[4])) {
			r.message = invalidAnchor
			slackPrint(r)
			return nil
		}
		if err := ChangeAnchor(chanTrim(words[2]), usrTrim(words[4])); err != nil {
			if err == ErrNoComponent {
				r.message = noComponentInDB
			} else if err == ErrNoChannel {
				r.message = noChannelInSlack
			} else {
				log.Panic(err)
			}
			slackPrint(r)
			return nil
		}
		r.message = fmt.Sprintf("Successfully changed anchor for %s to %s", words[2], words[4])
		slackPrint(r)
	case regAnchor.MatchString(words[1]):
		handleAnchor(ev, words[1:])

	default:
		handleKeywords(ev, words)

	}
	return nil
}

func (r *response) setResponseContext(ev *slack.MessageEvent) {
	chanInfo, err := sc.GetConversationInfo(ev.Channel, false)
	if err != nil {
		log.Error(err)
	}
	if !chanInfo.IsIM {
		if ev.ThreadTimestamp != "" {
			r.threadTS = ev.ThreadTimestamp
		} else {
			r.threadTS = ev.Timestamp
		}
	}

}

func tagCleanup(message string) []string {
	// Message like: "@bot tag #channel tag1, tag2a tag2b , tag3a b   tag3c"
	words := strings.Split(message, ",")
	for i, word := range words {
		words[i] = strings.Join(strings.Fields(strings.Trim(word, " ")), " ")
	}
	words[0] = strings.Join(strings.Fields(words[0])[3:], " ")
	return words
}
