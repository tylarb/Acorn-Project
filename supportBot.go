/*
Support bot - first response and basic AI for handling new support questions in Slack
for the Pivotal support team

PCF enabled - requires Postgres Database instance as well as bot integration key
provided by Slack

Released under MIT liscence, copyright 2018 Tyler Ramer
*/

package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/cloudfoundry-community/go-cfenv"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
)

const serviceLable = "elephantsql"

var conStr string

// Get bot name and token from env, and make sure botID is globally accessible
var (
	slackBotToken   = os.Getenv("SLACK_BOT_TOKEN")
	slackBotName    = os.Getenv("SLACK_BOT_NAME")
	slackBotChannel = os.Getenv("SLACK_BOT_CHANNEL")
	botID           string
	chanID          string
)

// The slack client and RTM messaging are used as an out - rather than passing
// the SC to each function, define it globally to ease accessed. We do handle
// errors in the main function, however

var (
	sc  *slack.Client
	rtm *slack.RTM
)

// log levels, see logrus docs
func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	appEnv, err := cfenv.Current()
	if err != nil {
		log.Fatal("Could not get cloud foundry environment details")
	}
	dbService, err := appEnv.Services.WithLabel(serviceLable)
	if err != nil {
		log.Fatal(err)
	}

	if len(dbService) != 1 {
		log.WithField("serviceLable", serviceLable).Fatal("It appears that more than one service with the serviceLable is attached to your app\nPlease ensure there is only one database instance attached")
	}
	var ok bool
	conStr, ok = dbService[0].CredentialString("uri")
	if !ok {
		log.Fatal("Could not find database URI")
	}
	InitDB()
	err = MigrateDB()
	if err != nil {
		log.Error("Could not query tables and had a problem creating them successfully, closing")
		log.Fatal(err)
	}
}

func main() {
	rand.Seed(time.Now().Unix())

	sc = slack.New(slackBotToken)
	botID = getBotID(slackBotName, sc)
	chanID = getBotChannel(slackBotChannel, sc)
	log.WithField("ID", botID).Debug("Bot ID returned")
	rtm = sc.NewRTM()
	go rtm.ManageConnection()
	log.Info("Connected to slack")

	for slackEvent := range rtm.IncomingEvents {
		switch ev := slackEvent.Data.(type) {
		case *slack.HelloEvent:
			// Ignored
		case *slack.ConnectedEvent:
			log.WithFields(log.Fields{"Connection Counter:": ev.ConnectionCount, "Infos": ev.Info})
		case *slack.MessageEvent:
			log.WithFields(log.Fields{"Channel": ev.Channel, "message": ev.Text}).Debug("message event:")
			// send message to parser func
			err := parse(ev)
			if err != nil {
				log.WithField("ERROR", err).Error("parse message failed")
			}
		case *slack.MemberJoinedChannelEvent:
			if ev.Channel == chanID {
				err := postHelpJoin(ev)
				if err != nil {
					log.Error("could not post help on user join channel")
					log.Error(err)
				}
			}

		case *slack.LatencyReport:
			log.WithField("Latency", ev.Value).Debug("Latency Reported")
		case *slack.RTMError:
			log.WithField("ERROR", ev.Error()).Error("RTM Error")
		case *slack.InvalidAuthEvent:
			log.Error("Invalid Credentials")
			return
		default:
			log.WithField("Data", ev).Debug("Some other data type")

		}

	}

}
