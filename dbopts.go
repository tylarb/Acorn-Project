/*
Contains database operations for the bot.


Released under MIT license, copyright 2018 Tyler Ramer

*/
package main

import (
	"database/sql"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// Table creation SQL strings
const (
	componentsTable = "CREATE TABLE IF NOT EXISTS components(id SERIAL PRIMARY KEY, component VARCHAR(20), slack_channel VARCHAR(10), playbook VARCHAR(30), anchor_id INT)"
	anchorsTable    = "CREATE TABLE IF NOT EXISTS anchors(id SERIAL PRIMARY KEY, anchor_slack VARCHAR(20), salesforceID VARCHAR(30), name VARCHAR(50)) "
	tagsTable       = "CREATE TABLE IF NOT EXISTS tags(id SERIAL PRIMARY KEY, tag VARCHAR(20), component_id INT)"
)

// TODO: make sure this makes sense
type tagInfo struct {
	name           string
	anchor         string
	component      string
	slackChannelID string
	playbook       string
}

// Define a global db connection. We don't need to close the db conn - if there's an error we'll try
// to recreate the db connection, but otherwise we don't intend to trash it
var db *sql.DB

// Connect to the DB and test the connection. Because we're using a global DB connection, and because
// database/sql will retry the connection for us, we should only use this to initialize the db connection
func dbConnect() *sql.DB {
	var err error
	db, err = sql.Open("postgres", conStr)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Error("Trouble connecting to the database, shutting down")
		log.Fatal(err)
	}
	log.WithField("conStr", conStr).Info("Successfully connected to a postgres DB")
	// TODO create relevant tables and check them
	//	checkTables()
	return db
}

// confirm all database tables exist and exit if they don't try to create them
func checkTables() error {

	var result string
	var err error
	err = db.QueryRow("SELECT 1 FROM components LIMIT 1").Scan(&result)

	if err != nil {
		log.Error("Could not select from component table, will attempt to create it now")
		err = createComponentsTable()
	}
	err = db.QueryRow("SELECT 1 from anchors LIMIT 1").Scan(&result)

	if err != nil {
		log.Error("Could not select from anchor table, will attempt to create it now")
		err = createAnchorsTable()
	}
	err = db.QueryRow("SELECT 1 from tags LIMIT 1").Scan(&result)

	if err != nil {
		log.Error("Could not select from tags table, will attempt to create it now")
		err = createTagsTable()
	}
	return err
}

// creates the "components" table in database
func createComponentsTable() error {
	_, err := db.Exec(componentsTable)
	if err != nil {
		log.Error("Problem creating components table")
		log.Fatal(err)
	}
	log.Info("Successfully created components table")
	return nil
}

// creates the "anchors" table in database
func createAnchorsTable() error {
	_, err := db.Exec(anchorsTable)
	if err != nil {
		log.Error("Problem creating anchors table")
		log.Fatal(err)
	}
	log.Info("Successfully created anchors table")
	return nil
}

// creates the "tags" table in database
func createTagsTable() error {
	_, err := db.Exec(tagsTable)
	if err != nil {
		log.Error("Problem creating tags table")
		log.Fatal(err)
	}
	log.Info("Successfully created tags table")
	return nil
}

// TODO: make this able to handle more than one component per tag (i.e. if tags return multiple components)
func keywordAsk(n string) (retTags []tagInfo) {
	var t tagInfo
	t.name = n
	var (
		componentID int
		anchorID    int
		rows        *sql.Rows
		err         error
	)

	//	err = db.QueryRow("SELECT component_id from tags WHERE tag=$1", n).Scan(&componentID)
	//	if err != nil {
	//		log.Fatal(err) // TODO: go ahead and exit if tag does exist
	//	}
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
