package main

import (
	"fmt"
	"testing"
)

func TestGetElasticsearchSessionError(t *testing.T) {
	cfg := &ElasticsearchData{
		URL:      "http://1.2.3.4:1234",
		Index:    "",
		Username: "",
		Password: "",
	}

	err := initElasticsearchSession(cfg)
	if err == nil {
		t.Error(fmt.Errorf("An error was expected while initializing ElasticSearch"))
	}
}
