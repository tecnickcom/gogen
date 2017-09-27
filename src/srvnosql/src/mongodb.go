package main

import (
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
func initMongodbSession(cfg *MongodbData) (*MongodbData, error) {
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
		return nil, err
	}
	cfg.session = mgosession
	return cfg, nil
}

// isMongodbAlive returns the status of MongoDB
func isMongodbAlive() error {
	session := appParams.mongodb.session.Copy()
	defer session.Close()
	_, err := session.DatabaseNames()
	return err
}
