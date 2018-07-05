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

	log "github.com/sirupsen/logrus"
	"github.com/tylarb/slack"
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
	askHelp    = regexp.MustCompile(`^(?i)help[\?]?$`)
	askKeyword = regexp.MustCompile(`^(?i)keyword[s]?[\:]?$`)
)

// posts a help message on user join
func postHelpJoin(ev *slack.MemberJoinedChannelEvent) error {
	message := `Hi! It looks like this is your first time joining this channel.
Please follow this guide for getting help from the bot:
type "keyword: [keyword]" to see playbooks, appropriate channels, and the anchor associated with this topic
type "anchor: [keyword]" to see the anchor and channel in charge of a product
type "help" in this channel to see this message again at any time`

	r := response{message, ev.User, ev.Channel, true, false, ""}
	err := slackPrint(r)
	if err != nil {
		log.Error("error printing to Slack")
	}
	return err
}

// posts a general help message on user asking for help in channel
func postHelp(ev *slack.MessageEvent) error {
	message := `Please follow this guide for getting help from the bot:
type "keyword: [keyword]" to see playbooks, appropriate channels, and the anchor associated with this topic
type "anchor: [keyword]" to see the anchor and channel in charge of a product
type "help" in this channel to see this message again at any time`

	r := response{message, ev.User, ev.Channel, true, false, ""}
	err := slackPrint(r)
	if err != nil {
		log.Error("error printing to Slack")
	}
	return err
}

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
		log.WithField("Message", ev.Text).Debug("Handling individual words")
		err = handleWord(ev, words)
	}

	return nil
}

// regex match and take appropriate action on words in a sentance. This only gets executed if
// the message is not deemed some other "type" of interation - like a command to the bot
func handleWord(ev *slack.MessageEvent, words []string) (err error) {
	switch {
	case askHelp.MatchString(words[0]):
		log.Debug("handling a help message")
		handleHelp(ev, words)
	case askKeyword.MatchString(words[0]):
		log.Debug("Handling a keyword message")
		if len(words) > 1 {
			handleKeywords(ev, words)
		} else {
			postHelp(ev)
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
	if len(words) == 1 {
		postHelp(ev)
	} // TODO add helps for adding to database
	return nil
}

// handlesKeywords passed via the "keyword" option
func handleKeywords(ev *slack.MessageEvent, words []string) error {
	// TODO: handle the printing better
	var t tagInfo
	for i := 1; i < len(words); i++ {
		t = keywordAsk(words[i])
		s := fmt.Sprintf("tag: %s, anchor: %s, component: %s, channel: %s, playbook: %s", t.name, t.anchor, t.component, t.slackChannelID, t.playbook)
		var r = response{s, ev.User, ev.Channel, false, false, ev.EventTimestamp}
		slackPrint(r)
	}
	return nil
}

// TODO: add Servicecloud integration and scan cases for details
func handleCase(ev *slack.MessageEvent, words []string) error {
	return nil
}

// Commands directed at the bot
func handleCommand(ev *slack.MessageEvent, words []string) error {

	return nil
}

// Print messages to slack. Accepts response struct and returns any errors on the print
func slackPrint(r response) (err error) {
	switch {
	case r.isEphemeral:
		_, err = postEphemeral(rtm, r.channel, r.user, r.message)
	default:
		rtm.SendMessage(rtm.NewOutgoingMessage(r.message, r.channel, r.threadTS))
		err = nil
	}
	return
}

// formats the user string to make sure indidual gets tagged correctly in slack
func usrFormat(u string) string {
	return fmt.Sprintf("<@%s>", u)
}
