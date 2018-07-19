/*
All help and error mesages which might be printed to slack
*/

package main

import (
	"fmt"

	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

const (
	baseHelp = iota
	tagsHelp
	addHelp
)

// Various help messages
const (
	noComponentInDB = "This component is not in the database - please reach out to a member of acorn project team to get your component added"
	noRelevantTag   = "I couldn't find anything relevant. Please contact your local (or remote) anchor if you thing you have a tag which should be added"
	alreadyAdded    = "Tag _%s_ is already marked for this component"
)

func tagFmt(tag TagInfo) string {
	return fmt.Sprintf("*tag:* %s, *anchor:* %s, *component-channel:* %s, *support-channel:* %s, *playbook:* %s\n", tag.Name, usrFormat(tag.Anchor), chanFormat(tag.ComponentChan), chanFormat(tag.SupportChan), tag.PlaybookURL)
}

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

_@[bot] tag [#component-channel] [tag1], [tag2], ..._

Only anchors can add tags.`

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
