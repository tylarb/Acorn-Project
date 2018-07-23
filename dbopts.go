/*
dbopts.go contains database operations for the bot.

We contemplate three data models: anchor, component and tag. The TagInfo struct
is used to bundle the response information that will be displayed by slackParse
logic.

Released under MIT license, copyright 2018 Tyler Ramer, Ignacio Elizaga
*/
package main

import (
	"errors"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	log "github.com/sirupsen/logrus"
)

// Component is the database representation of a component
type Component struct {
	ID            int
	AnchorSlackID string `gorm:"type:varchar(20)"`
	PlaybookURL   string `gorm:"type:varchar(100)"`
	ComponentChan string `gorm:"type:varchar(20)"`
	SupportChan   string `gorm:"type:varchar(20)"`
}

// Tag is the database representation of a tag
type Tag struct {
	ID         int
	Name       string      `gorm:"type:varchar(20)"`
	Components []Component `gorm:"many2many:tag_components;"`
}

var db *gorm.DB

// InitDB creates the connection to the database specified in conStr and stores
// it in the db local variable
func InitDB() {
	var err error
	db, err = gorm.Open("postgres", conStr)
	if err != nil {
		log.Error("Trouble connecting to the database, shutting down")
		log.Panic(err)
	}
	log.WithField("conStr", conStr).Info("connected to the database")
}

// MigrateDB performs a database migration from scratch for any of the db tables
// if they don't exist. This does not include ddl changes in existing tables
func MigrateDB() error {
	var err error
	if err = db.AutoMigrate(&Component{}, &Tag{}).Error; err != nil {
		log.Error("the migration has failed")
		log.Panic(err)
	}
	return err
}

// QueryTag scans the database for a given tag name and returns a slice of
// TagInfo objects
func QueryTag(n string) (retTags []TagInfo, err error) {
	var (
		t          TagInfo
		tag        Tag
		components []Component
	)

	// query the tag
	if err := db.Where("Name = ?", n).First(&tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			log.WithField("tag", n).Error("tag not found")
			// TODO:  we can add some logic to this and ask the user to notify an
			// anchor
			return nil, ErrNoTag
		}
		log.Error("an error ocurred querying the database for tag")
		log.Panic(err)
	}

	// query the components associated with the tag
	if err := db.Model(&tag).Association("Components").Find(&components).Error; err != nil {
		log.Error("an error ocurred querying the database for components associated with tag")
		log.Panic(err)
	}

	t.Name = n // More than one component for some tags, but this method handles a single tag name
	for _, component := range components {
		t.Anchor = component.AnchorSlackID
		t.ComponentChan = component.ComponentChan
		t.PlaybookURL = component.PlaybookURL
		t.SupportChan = component.SupportChan
		retTags = append(retTags, t)
	}
	log.WithField("retTags[]", retTags).Info("tag information found")

	return
}

// GetAllTags retrives all tags in the database into a map for use in the cache
func GetAllTags() (tagMap map[string][]TagInfo, size int) {
	var (
		t          TagInfo
		tags       []Tag
		components []Component
	)
	tagMap = make(map[string][]TagInfo)

	if err := db.Find(&tags).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			log.Info("Currently no tags to load from the database")
			return nil, 0
		}
		log.Panic(err)
	}
	for _, tag := range tags {
		if err := db.Model(&tag).Association("Components").Find(&components).Error; err != nil {
			log.Error("An error occured querying the database for components associated with a tag")
			log.Panic(err)
		}
		var retTags []TagInfo
		t.Name = tag.Name
		for _, component := range components {
			t.Anchor = component.AnchorSlackID
			t.ComponentChan = component.ComponentChan
			t.PlaybookURL = component.PlaybookURL
			t.SupportChan = component.SupportChan
			retTags = append(retTags, t)
		}
		tagMap[t.Name] = retTags
		log.WithFields(log.Fields{"name": t.Name, "tagInfo": retTags}).Debug("tag information retrieved from database")
	}
	size = len(tags)
	log.WithField("number", size).Info("Tags returned from the database")
	return
}

// AddTag adds a component tag to the database
// Note: in usage, query the cache for a tag before going to DB; thus, checking if tag
// entry already exists should not be an issue here
// TODO: Eventually, adding a component may be possible. Need to build error logic if
// component exists w/ different information than provided. Also need to clean up probably
func AddTag(t TagInfo) error {
	var component Component
	tag := Tag{Name: t.Name}

	if err := db.Where(&Component{ComponentChan: t.ComponentChan}).First(&component).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			// Check if the channel exists in slack
			channel, err := getChanName(t.ComponentChan)
			if err != nil {
				log.WithField("ComponentChannel", t.ComponentChan).Error("Component channel is not valid")
				return err
			}
			log.WithField("ComponentName", channel).Error("Component is not in the DB")
			return ErrNoComponent
		}
		log.Panic(err)
	}

	if err := db.Where(&tag).First(&tag).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			log.WithField("tag", tag.Name).Info("Adding new tag to DB")
		} else {
			log.Error("an error ocurred querying the database for tag")
			log.Panic(err)
		}
	}

	if db.NewRecord(tag) {
		tag.Components = append(tag.Components, component)
		if err := db.Create(&tag).Error; err != nil {
			log.Error("Failed creating a tag in the database")
			log.Panic(err)
		}
	} else {
		if err := db.Model(&tag).Association("Components").Append(component).Error; err != nil {
			log.Error("Failed adding a tag association to the database")
			log.Panic(err)
		}
	}
	supportChan, _ := getChanName(component.SupportChan)
	componentChan, _ := getChanName(component.ComponentChan)
	log.WithFields(log.Fields{"tag": t.Name, "support-channel": supportChan, "component-channel": componentChan}).Info("added tag to the database")
	return nil
}

// ErrNoComponent is returned of there is no component in DB with provided ID
var ErrNoComponent = errors.New("No component returned from the database for this ID")

// ErrNoTag is returned if there is no tag in the DB for the associated entry TODO - add error to be returned by the cache
var ErrNoTag = errors.New("No tag exists for this word")
