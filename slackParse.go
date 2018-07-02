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
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"github.com/tylarb/TimeCache"
)

// set URL for expansion here
// TODO: change to OS env variable and/or move to SDFC api
const baseURL = "http://example.com/"

// Timeout in seconds to prevent karma spam
const timeout = 5 * 60

type response struct {
	message     string
	user        string
	channel     string
	isEphemeral bool
	isIM        bool
}

type karmaVal struct {
	name    string
	points  int
	shame   bool
	present bool // is the name present in the database
}

type cacheKey struct {
	*string
}

var cache = timeCache.NewSliceCache(timeout)



// parses all messagess from slack for special commands or karma events
func parse(ev *slack.MessageEvent) (err error) {
	var atBot = fmt.Sprintf("<@%s>", botID)
	if ev.User == "USLACKBOT" {
		log.Debug("Slackbot sent a message which is ignored")
		return nil
	}
	words := strings.Split(ev.Text, " ")
	switch {
	case words[0] == atBot:
		log.WithField("Message", ev.Text).Debug("Instuction for bot")
		err = handleCommand(ev, words)
	default:
		err = handleWord(ev, words)
	}

	return nil
}

// regex match and take appropriate action on words in a sentance. This only gets executed if
// the message is not deemed some other "type" of interation - like a command to the bot
func handleWord(ev *slack.MessageEvent, words []string) (err error) {

	var (
		s       string
		count   int
		message string
		k       *karmaVal
		key     string    // key = user + target to prevent vote spam
		tc      bool      // time key was added to the cache
		r       time.Time // time remaining until able to be upvoted
	)

	retArray := []string{}
	caseLinks := []string{}


	var retMessage = response{message, ev.User, ev.Channel, false, false}
	err = slackPrint(retMessage)
	if err != nil {
		log.WithField("Err", err).Error("unable to print message to slack")
		return err
	}
	return nil

}

// Commands directed at the bot
func handleCommand(ev *slack.MessageEvent, words []string) error {
	retArray := []string{}
	var message string
	var s string
	var err error
	var k *karmaVal
	var rank int

	switch {
	case len(words) > 2 && words[1] == "rank": // individual karma rankings

	return nil
}

// Print messages to slack. Accepts response struct and returns any errors on the print
func slackPrint(r response) (err error) {
	switch {
	case r.isEphemeral:
		_, err = postEphemeral(rtm, r.channel, r.user, r.message)
	default:
		rtm.SendMessage(rtm.NewOutgoingMessage(r.message, r.channel))
		err = nil
	}
	return
}

func timeWarn(ev *slack.MessageEvent, n string, t time.Time) {
	tRemain := time.Duration(timeout)*time.Second - time.Since(t)
	message := fmt.Sprintf("Please wait %v before adjusting the karma of %s", tRemain, n)
	var r = response{message, ev.User, ev.Channel, true, false}
	slackPrint(r)
}

func newKarma(name string, shame bool) *karmaVal {
	k := new(karmaVal)
	k.name = name
	k.shame = shame
	return k
}

func keygen(u string, t string) string {
	s := []string{u, t}
	k := strings.Join(s, "-")
	return k
}

func responseGen(k *karmaVal, rank int) (s string) {
	if rank != 0 {
		switch {
		case k.shame && k.points == 1:
			s = fmt.Sprintf("What is done cannot be undone. %s now has shame forever\n", k.name)
		case k.shame:
			s = fmt.Sprintf("%s now has %d points of shame\n", k.name, k.points)
		default:
			s = fmt.Sprintf("%s now has %d points of karma\n", k.name, k.points)
		}
	} else {
		if k.shame {
			s = fmt.Sprintf("%s is rank %d with %d points of shame\n", k.name, rank, k.points)
		} else {
			s = fmt.Sprintf("%s is rank %d with %d points of shame\n", k.name, rank, k.points)
		}
	}
	return
}

func usrFormat(u string) string {
	return fmt.Sprintf("<@%s>", u)
}
