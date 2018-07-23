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

// regex definitions

var (
	regHelp = regexp.MustCompile(`^(?i)help[\?]?$`)
	regTags = regexp.MustCompile(`^(?i)tag[s]?[\:]?$`)
	regAdd  = regexp.MustCompile(`(?i)add$`)
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
	}

	return nil
}

// handlesKeywords passed via the "tag" option
func handleKeywords(ev *slack.MessageEvent, words []string) error {
	var (
		responses        []string
		s                string // Placeholder string for building a response
		r                response
		foundMatch       bool
		matchDistPercent = .75
		minWordLength    = 4
	)

	wordCache := cache.GetNames()

	for i := 1; i < len(words); i++ {
		if len(words[i]) < minWordLength {
			if cache.ContainsTag(words[i]) {
				foundMatch = true
				for _, tag := range cache.Find(words[i]) {
					s = tagFmt(tag)
					responses = append(responses, s)
				}
			}
		} else {
			for _, tag := range wordCache {
				dist := lv.DistanceForStrings([]rune(words[i]), []rune(tag), lv.DefaultOptions)
				lWord := float64(len(words[i]))
				d := float64(dist)
				lTag := float64(len(tag))
				log.WithFields(log.Fields{"s1": tag, "s2": words[i], "dist": dist}).Debug("Levenshtein distance")
				if dist == 0 {
					foundMatch = true
					for _, tag := range cache.Find(words[i]) {
						s = tagFmt(tag)
						responses = append(responses, s)
					}
				} else if lWord-d/lTag > matchDistPercent {
					foundMatch = true
					for _, tag := range cache.Find(words[i]) {
						s = tagFmt(tag)
						responses = append(responses, s)
					}
				}

			}
		}
	}
	if foundMatch {
		s = strings.Join(responses[:], "\n")
	} else {
		s = noRelevantTag
	}
	r = response{s, ev.User, ev.Channel, false, false, ev.EventTimestamp}
	slackPrint(r)
	return nil

}

// TODO: add Servicecloud integration and scan cases for details
func handleCase(ev *slack.MessageEvent, words []string) error {
	return nil
}

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
		for i := 3; i < len(words); i++ {
			tag.Name = strings.Trim(words[i], ",")
			if !cache.ContainsTagInfo(tag) {
				if err := cache.Add(tag); err != nil {
					if err == ErrNoComponent {
						r.message = noComponentInDB
						slackPrint(r)
						break
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
	default:
		handleKeywords(ev, words)

	}
	return nil
}
