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

const (
	baseHelp = iota
	tagsHelp
	addHelp
)

// posts a help message on user join
func postHelpJoin(ev *slack.MemberJoinedChannelEvent) error {
	message := `Hi! It looks like this is your first time joining this channel.
Please follow this guide for getting help from the bot:

type _tag: [keyword]_ to see the component, playbooks, appropriate channels, and the anchor associated with this tag

type _anchor: [component]_ to see the anchor and channel in charge of a product component 

type _help_ in this channel to see this message again at any time`

	r := response{message, ev.User, ev.Channel, true, false, ""}
	err := slackPrint(r)
	if err != nil {
		log.Error("error printing to Slack")
	}
	return err
}

// posts a general help message on user asking for help in channel
func postHelp(ev *slack.MessageEvent, kind int) error {
	var message string
	switch {
	case kind == baseHelp:
		message = `type _tag: [keyword]_ to see component, playbooks, appropriate channels, and the anchor associated with this tag

type _anchor: [component]_ to see the anchor and channel in charge of a product

type _help_ in this channel to see this message again at any time

type _help tags_ for further information about adding tags

type _help add_ for help adding other details to the database`
	case kind == tagsHelp:
		message = `To add tags to the bot, use the following syntax:
	
_@[bot] tag [keyword] as [component name]_

Only anchors can add tags. To see a list of valid component names type:
_@[bot] list components_`

	case kind == addHelp:
		message = `Adding other items to the database is still in development. Check back later`
	}

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
	// TODO: handle the printing better
	var tags []tagInfo
	var responses []string
	var s string
	var r response
	for i := 1; i < len(words); i++ {
		tags = keywordAsk(words[i])
		fmt.Printf("%v\n", tags)
		if len(tags) == 0 {
			s = fmt.Sprintf("There are no components associated with the tag %s - please contact an anchor if you believe this tag should be added", words[i])
			responses = append(responses, s)
		} else {
			for _, tag := range tags {
				s := fmt.Sprintf("tag: %s, anchor: %s, component: %s, channel: %s, playbook: %s\n", tag.name, usrFormat(tag.anchor), tag.component, chanFormat(tag.slackChannelID), tag.playbook)
				responses = append(responses, s)
			}
		}
	}
	s = strings.Join(responses[:], " ")
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
	switch {
	case regTags.MatchString(words[1]):
		if len(words) < 5 {
			postHelp(ev, tagsHelp)
		} //TODO complete this - Tag component Slack Chan as the ID - import ID to database via strings.Trim(c,"<>") then cut on | and accept first array value
	case regAdd.MatchString(words[1]):
		switch {
		case len(words) < 5:
			postHelp(ev, addHelp)
		case words[2] == "component":
			postHelp(ev, addHelp) // TODO finish building matrix/DB to add component
		case words[4] == "as":
			postHelp(ev, addHelp) // TODO finish building matrix/DB to add anchor
		}

	}
	return nil
}

// Print messages to slack. Accepts response struct and returns any errors on the print
func slackPrint(r response) (err error) {
	switch {
	case r.isEphemeral:
		_, err = postEphemeral(r.channel, r.user, r.message)
	default:
		rtm.SendMessage(rtm.NewOutgoingMessage(r.message, r.channel, slack.RTMsgOptionTS(r.threadTS)))
		err = nil
	}
	return
}

// formats the user string to make sure indidual gets tagged correctly in slack
func usrFormat(u string) string {
	return fmt.Sprintf("<@%s>", u)
}

// formats a channel ID to allow channel linking update to slack
func chanFormat(c string) string {
	return fmt.Sprintf("<#%s>", c)
}
