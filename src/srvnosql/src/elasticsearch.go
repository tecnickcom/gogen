package main

import (
	"context"
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
func initElasticsearchSession(cfg *ElasticsearchData) (*ElasticsearchData, error) {

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

	return cfg, err
}

// isElasticsearchAlive returns the status of ElasticSearch
func isElasticsearchAlive() error {
	_, code, err := appParams.elasticsearch.client.Ping(appParams.elasticsearch.URL).Do(appParams.elasticsearch.ctx)
	if err == nil && code != 200 {
		err = fmt.Errorf("invalid response code: %d", code)
	}
	return err
}
