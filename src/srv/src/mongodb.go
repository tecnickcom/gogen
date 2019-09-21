package main

import (
	"errors"
	"time"

	"github.com/globalsign/mgo"
	log "github.com/sirupsen/logrus"
	//"github.com/globalsign/mgo/bson"
)

// MongodbData store a single MongoDB configuration
type MongodbData struct {
	Address  string       `json:"address"`  // MongoDB address: [mongodb://][user:pass@]host1[:port1][,host2[:port2],...][/database][?options]
	Database string       `json:"database"` // MongoDB database name.
	User     string       `json:"user"`     // MongoDB user name.
	Password string       `json:"password"` // MongoDB password.
	Timeout  int          `json:"timeout"`  // MongoDB connection timeout.
	session  *mgo.Session // MongoDB session
}

// initMongodbSession return a new MongoDB session
func initMongodbSession(cfg *MongodbData) error {
	if cfg.Address == "" {
		cfg.session = nil
		return nil
	}
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    []string{cfg.Address},
		Timeout:  time.Duration(cfg.Timeout) * time.Second,
		Database: cfg.Database,
		Username: cfg.User,
		Password: cfg.Password,
	}
	//mgo.SetLogger(stdLogger)
	mgosession, err := mgo.DialWithInfo(mongoDBDialInfo)
	if err != nil {
		log.WithFields(log.Fields{
			"error":          err,
			"mongodbAddress": cfg.Address,
		}).Error("Unable to connect to the MongoDB server")
		return err
	}
	cfg.session = mgosession
	return nil
}

// Close the MongoDB connection
func (mg *MongodbData) Close() {
	if mg.session == nil {
		return
	}
	mg.session.Close()
}

// isMongodbAlive returns the status of MongoDB
func isMongodbAlive() error {
	if appParams.mongodb.session == nil {
		return errors.New("mongodb is not available")
	}
	session := appParams.mongodb.session.Copy()
	defer session.Close()
	_, err := session.DatabaseNames()
	return err
}
