package main

import (
	"testing"
)

func TestGetMongodbSessionError(t *testing.T) {
	cfg := &MongodbData{
		Address:  "1.2.3.4:4321",
		Database: "",
		User:     "",
		Password: "",
		Timeout:  1,
	}

	err := initMongodbSession(cfg)
	if err == nil {
		t.Errorf("An error was expected while initializing MongoDB")
	}
}
