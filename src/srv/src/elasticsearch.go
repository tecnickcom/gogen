package main

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/olivere/elastic.v5"
	"gopkg.in/olivere/elastic.v5/config"
)

// ElasticsearchData store a single ElasticSearch configuration
type ElasticsearchData struct {
	URL      string `json:"url"`      // ElasticSearch URL
	Index    string `json:"index"`    // ElasticSearch main index
	Username string `json:"username"` // ElasticSearch user name
	Password string `json:"password"` // ElasticSearch password
	ctx      context.Context
	client   *elastic.Client
}

// initElasticsearchSession return a new ElasticSearch session
func initElasticsearchSession(cfg *ElasticsearchData) error {
	if cfg.URL == "" {
		cfg.client = nil
		return nil
	}
	ecfg := &config.Config{
		URL:      cfg.URL,
		Index:    cfg.Index,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	client, err := elastic.NewClientFromConfig(ecfg)
	if err == nil {
		ctx := context.Background()
		cfg.ctx = ctx
		cfg.client = client
	}
	return err
}

// isElasticsearchAlive returns the status of ElasticSearch
func isElasticsearchAlive() error {
	if appParams.elasticsearch.client == nil {
		return errors.New("elasticsearch is not available")
	}
	_, code, err := appParams.elasticsearch.client.Ping(appParams.elasticsearch.URL).Do(appParams.elasticsearch.ctx)
	if err == nil && code != 200 {
		err = fmt.Errorf("invalid response code: %d", code)
	}
	return err
}
