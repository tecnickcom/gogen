package main

import (
	"fmt"
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

	_, err := initMongodbSession(cfg)
	if err == nil {
		t.Error(fmt.Errorf("An error was expected while initializing MongoDB"))
	}
}
