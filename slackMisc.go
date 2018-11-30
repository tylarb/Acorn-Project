/*
Various addons and handlers for slack to make the API easier to use.


Released under MIT license, copyright 2018 Tyler Ramer

*/

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

// getBotChannel gets a userID from a username provided as a string. This is easy to provide as a startup option or env variable.
// For further consideration - can this be done using ONLY a token?
func getBotID(botName string, sc *slack.Client) (botID string) {
	users, err := sc.GetUsers()
	if err != nil {
		log.Fatal(err)
	}
	for _, user := range users {
		if user.Name == botName {
			log.WithFields(log.Fields{"ID": user.ID, "name": user.Name}).Info("Found bot:")
			botID = user.ID
			return
		}
	}
	log.Fatal("Could not find a userID for the botID provided")
	return
}

// getBotChannel gets a chanID from a channel name provided as a string. This is easy to provide as a startup option or env variable.
func getBotChannel(chanName string, sc *slack.Client) (chanID string) {
	channels, err := sc.GetChannels(true)
	if err != nil {
		log.Fatal(err)
	}
	for _, channel := range channels {
		if channel.Name == chanName {
			log.WithField("Channel ID", channel.ID).Info("Found channel")
			chanID = channel.ID
			return
		}
	}
	log.Fatal("Could not find a channelID for the channel name provided")
	return
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

// trims a slack channel provided in format <#CBJJ3CUAZ|anchorchan2> to just the chanID
func chanTrim(c string) string {
	s := strings.Split(c, "|")
	r := strings.Trim(s[0], "<#")
	return r
}

func usrTrim(u string) string {
	return strings.Trim(u, "<@>")
}

// gets a channel name from ID via API for cleaner printing to logs
func getChanName(id string) (string, error) {
	channel, err := sc.GetChannelInfo(id)
	if err != nil {
		log.WithField("id", id).Error("API call to get chan info failed")
		if err.Error() == "channel_not_found" {
			return "", ErrNoChannel
		}
		log.Panic(err)
	}
	return channel.Name, nil
}

func validateAnchorName(n string) bool {
	_, err := sc.GetUserInfo(n)
	if err != nil {
		return false
	}
	return true

}

// Cleans up Ephemeral message posting, see issue: https://github.com/nlopes/slack/issues/191
func postEphemeral(channel, user, text string) (string, error) {
	params := slack.PostMessageParameters{
		AsUser: true,
	}
	return rtm.PostEphemeral(
		channel,
		user,
		slack.MsgOptionText(text, params.EscapeText),
		slack.MsgOptionAttachments(params.Attachments...),
		slack.MsgOptionPostMessageParameters(params),
	)
}

// ErrNoChannel is returned if there is no channel in slack with this name
var ErrNoChannel = errors.New("No channel exists in slack with this name")
