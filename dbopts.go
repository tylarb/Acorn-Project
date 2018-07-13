/*
Contains database operations for the bot.

Released under MIT license, copyright 2018 Tyler Ramer
*/
package main

import (
	"database/sql"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type Component struct {
	ID           int
	Anchor       int
	name         string `gorm:"type:varchar(20)"`
	Playbook     string `gorm:"type:varchar(30)"`
	SlackChannel string `gorm:"type:varchar(10)"`
}

type Anchor struct {
	ID           int
	name         string `gorm:"type:varchar(30)"`
	SalesforceID string `gorm:"type:varchar(30)"`
	SlackID      string `gorm:"type:varchar(20)"`
}

type Tag struct {
	ID         int
	name       string      `gorm:"type:varchar(20)"`
	components []Component `gorm:"many2many:TagComponent;"`
}

// TODO: make sure this makes sense
type tagInfo struct {
	anchor         string
	component      string
	name           string
	playbook       string
	slackChannelID string
}

var db *gorm.DB

// dbConnect connects to the database definied Connect to the DB and test the connection. Because we're using a global DB connection, and because
// database/sql will retry the connection for us, we should only use this to initialize the db connection
func dbConnect() *gorm.DB {
	db, err := gorm.Open("postgres", conStr)
	if err != nil {
		log.Fatal(err)
		// FIXME: Do we want to panic in this scenario? Also in checkTables().
		// panic("failed to connect database")
	}

	log.WithField("conStr", conStr).Info("Successfully connected to a postgres DB")
	return db
}

// CheckTables confirm all database tables exist and exit if they don't try to create them
func checkTables() error {

	// FIXME: Can we replace this logic with GORM autoMigrate function and handle errors?
	// Automigrate will only create the tables if they do not exist and will also create new columns that did not exist
	// before, but it will never drop columns that no longer exist
	// db.AutoMigrate(&Anchor{}, &Component{}, &Tag{})

	var err error

	if !db.HasTable(&Anchor{}) {
		log.Error("Anchors table does not exist, will attempt to create it now")
		if err := db.CreateTable(&Anchor{}).Error; err != nil {
			log.Error("Failed to create anchors table")
			// FIXME: We are printing err multiple times as this is printed again in SupportBot.go:68
			log.Fatal(err)
		}
	}

	// FIXME: does hasTable throw an error if the table is not found?
	if !db.HasTable(&Component{}) {
		log.Error("Components table does not exist, will attempt to create it now")
		if err := db.CreateTable(&Component{}).Error; err != nil {
			log.Error("Failed to create components table")
			log.Fatal(err)
		}
	}

	if !db.HasTable(&Tag{}) {
		log.Error("Tags table does not exist, will attempt to create it now")
		if err := db.CreateTable(&Tag{}).Error; err != nil {
			log.Error("Failed to create tags table")
			log.Fatal(err)
		}
	}

	if !db.HasTable("TagComponent") {
		log.Error("TagComponent table does not exist, will attempt to create it now")
		if err := db.CreateTable("TagComponent").Error; err != nil {
			log.Error("Failed to create tag-component table")
			log.Fatal(err)
		}
	}

	return err
}

// FIXME: make this able to handle more than one component per tag (i.e. if tags return multiple component)
// this should be easy with the many2many relationship defined in the Tag struct
func keywordAsk(n string) (retTags []tagInfo) {
	var (
		t           tagInfo
		componentID int
		anchorID    int
		rows        *sql.Rows
		err         error
	)

	t.name = n

	rows, err = db.Query("SELECT component_id FROM tags WHERE tag=$1", n)
	if err != nil {
		log.Error("problem selecting component_id from DB")
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&componentID)
		if err != nil {
			log.Error("problem scanning component ID")
			log.Fatal(err)
		}
		err = db.QueryRow("SELECT component,anchor_id,slack_channel,playbook FROM components WHERE id=$1", componentID).Scan(&t.component, &anchorID, &t.slackChannelID, &t.playbook) //TODO: edit if we restruct components table to use slackchan as comp ID

		if err != nil {
			log.Fatal(err) // THIS SHOULD NOT EVER CAUSE AN ISSUE - make sure component exists created when a tag gets created
		}
		err = db.QueryRow("SELECT anchor_slack FROM anchors WHERE id=$1", anchorID).Scan(&t.anchor)
		if err != nil {
			log.Fatal(err) // see note above - anchors are mandatory field in component
		}
		retTags = append(retTags, t)
	}
	return
}
