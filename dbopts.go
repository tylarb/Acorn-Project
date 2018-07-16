/*
dbopts.go contains database operations for the bot.

We contemplate three data models: anchor, component and tag. The TagInfo struct
is used to bundle the response information that will be displayed by slackParse
logic.

Released under MIT license, copyright 2018 Tyler Ramer, Ignacio Elizaga
*/
package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
)

// Component is the database representation of a component
type Component struct {
	ID            int
	AnchorSlackID string `gorm:"type:varchar(20)"`
	PlaybookURL   string `gorm:"type:varchar(30)"`
	ComponentChan string `gorm:"type:varchar(20)"`
}

// Tag is the database representation of a tag
type Tag struct {
	ID         int
	Name       string      `gorm:"type:varchar(20)"`
	Components []Component `gorm:"many2many:tag_components;"`
}

// TagInfo is the response structure when a tag query is made
type TagInfo struct {
	Anchor        string
	Name          string
	PlaybookURL   string
	ComponentChan string
}

// FIXME: This project structure might become hard to maintain as our database
// models grow, we might want to store each model and related database logic in
// a different file. That would add complexity to the code on the other side.

var db *gorm.DB

// InitDB creates the connection to the database specified in conStr and stores
// it in the db local variable
func InitDB() {
	var err error
	db, err = gorm.Open("postgres", conStr)
	if err != nil {
		log.Error("Trouble connecting to the database, shutting down")
		log.Fatal(err)
	}
	log.WithField("conStr", conStr).Info("connected to the database")
}

// MigrateDB performs a database migration from scratch for any of the db tables
// if they don't exist. This does not include ddl changes in existing tables
func MigrateDB() error {
	var err error
	if err = db.AutoMigrate(&Component{}, &Tag{}).Error; err != nil {
		log.Error("the migration has failed")
		log.Fatal(err)
	}
	return err
}

// QueryTag scans the database for a given tag name and returns a slice of
// TagInfo objects
func QueryTag(n string) (retTags []TagInfo) {
	t := TagInfo{}
	tag := Tag{}
	components := []Component{}

	// query the tag
	if err := db.Where("Name = ?", n).First(&tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			log.WithField("tag", n).Error("tag not found")
			// TODO:  we can add some logic to this and ask the user to notify an
			// anchor
			return
		}
		log.Error("an error ocurred querying the database for tag")
		log.Fatal(err)
	}

	// query the components associated with the tag
	if err := db.Model(&tag).Association("Components").Find(&components).Error; err != nil {
		log.Error("an error ocurred querying the database for components associated with tag")
		log.Fatal(err)
	}

	t.Name = n
	for _, component := range components {
		t.Anchor = component.AnchorSlackID
		t.ComponentChan = component.ComponentChan
		t.PlaybookURL = component.PlaybookURL
		retTags = append(retTags, t)
	}
	log.WithField("retTags[]", retTags).Info("tag information found")

	return
}
