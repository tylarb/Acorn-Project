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
	peopleTable = "CREATE TABLE IF NOT EXISTS people(id SERIAL PRIMARY KEY, name TEXT, karma INTEGER, shame INTEGER);"
	alsoTable   = "CREATE TABLE IF NOT EXISTS isalso(id SERIAL PRIMARY KEY, name TEXT, also TEXT);"
)

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

	// go ahead and check tables here
	checkTables()
	return db
}

// confirm all database tables exist and exit if they don't try to create them
func checkTables() {

	var result string
	err := db.QueryRow("SELECT 1 FROM people LIMIT 1").Scan(&result)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("Could not select from people table, will try to create it now")
			createPeopleTable()
		} else {
			log.Fatal(err)
		}
	}
	err = db.QueryRow("SELECT 1 from isalso LIMIT 1").Scan(&result)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("Could not select from isalso table, will try to create it now")
			createAlsoTable()
		} else {
			log.Fatal(err)
		}
	}

}

// creates the "people" table in database
func createPeopleTable() {
	_, err := db.Exec(peopleTable)
	if err != nil {
		log.Error("Problem creating people table")
		log.Fatal(err)
	}
}

// creates the "isalso" table in database
func createAlsoTable() {
	_, err := db.Exec(alsoTable)
	if err != nil {
		log.Error("Problem creating isalso table")
		log.Fatal(err)
	}
}

// karmaVal.ask accepts k karmaval and returns a karmaVal with k.points updated
func (k *karmaVal) ask() {

	var result int
	var err error
	var present = true
	if k.shame {
		err = db.QueryRow("SELECT shame FROM people WHERE name=$1", k.name).Scan(&result)
	} else {
		err = db.QueryRow("SELECT karma FROM people WHERE name=$1", k.name).Scan(&result)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			log.WithField("name", k.name).Debug("No karma or shame for this user yet")
			result = 0
			present = false
		} else {
			log.Fatal(err)
		}
	}
	k.points = result
	k.present = present
}

// karmaVal.rank returns the overall ranking of an individual entered as an INT. Does not
// return the actual karma of the indivitual, or 0 if user is not in DB
func (k *karmaVal) rank() int {
	var result int
	var err error
	if k.shame {
		err = db.QueryRow("SELECT (SELECT COUNT(*) FROM people AS t2 WHERE t2.shame > t1.shame) AS row_Num FROM people as t1 WHERE name=$1", k.name).Scan(&result)
	} else {
		err = db.QueryRow("SELECT (SELECT COUNT(*) FROM people AS t2 WHERE t2.karma > t1.karma) AS row_Num FROM people as t1 WHERE name=$1", k.name).Scan(&result)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return result
		}
		log.Error("issue getting RANK from the people database")
		log.Fatal(err)
	}
	return result + 1
}

// Handles moving karma or shame up or down. Accepts k karmaVal with k.name entered and an up/down
// flag, then returns updated karmaVal struct with points updated.
func (k *karmaVal) modify(upvote bool) {
	var err error
	k.ask()
	if upvote {
		k.points++
	} else {
		k.points--
	}
	if k.present {
		if k.shame {
			_, err = db.Exec("UPDATE people SET shame = $1 WHERE name = $2", k.points, k.name)
		} else {
			_, err = db.Exec("UPDATE people SET karma = $1 WHERE name = $2", k.points, k.name)
		}
		if err != nil {
			log.Error("There was a problem updating karma table in the database")
			log.Fatal(err)
		}
		log.WithField("Name", k.name).Debug("updated karma")
	} else {
		if k.shame {
			_, err = db.Exec("INSERT INTO people(name,karma,shame) VALUES($1,0,$2)", k.name, k.points)
		} else {
			_, err = db.Exec("INSERT INTO people(name,karma,shame) VALUES($1,$2,0)", k.name, k.points)
		}
		if err != nil {
			log.Error("There was an error inserting into karma table in the database")
			log.Fatal(err)
		}
		log.WithField("Name", k.name).Debug("inserted karma")
	}
}

//TODO Finish this
func globalRank(kind string) {
	var (
		name  string
		karma string
		err   error
		rows  *sql.Rows
	)
	switch {
	case kind == "top":
		rows, err = db.Query("SELECT name, karma FROM people ORDER BY karma DESC LIMIT 5")
		if err != nil {
			log.Error("Error selecting top karma from DB")
			log.Fatal(err)
		}
	case kind == "bottom":
	case kind == "shame":
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&name, &karma)
		if err != nil {
			log.Fatal(err)
		}

	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

}

// isAlsoAsk queries isAlso table in DB for a random entry of the inputed name, n. Returns empty string
// if value is not in the table
func isAlsoAsk(n string) string {
	var err error
	var result string
	err = db.QueryRow("SELECT also FROM isalso WHERE name=$1 ORDER BY RANDOM() LIMIT 1", n).Scan(&result)
	if err != nil {
		if err == sql.ErrNoRows {
			return ""
		}
		log.Error("There wa some error selecting value from the Also table")
		log.Fatal(err)
	}
	return result
}

// isAlsoAdd adds an "also" value to the database
func isAlsoAdd(n string, also string) {
	var err error
	_, err = db.Exec("INSERT INTO isalso(name,also) VALUES ($1,$2)", n, also)
	if err != nil {
		log.Error("There was an error inserting into the isalso table")
		log.Fatal(err)
	}
	log.WithField("name", n).WithField("value", also).Debug("Added to also table")
}
