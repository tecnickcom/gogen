package main

import (
	"context"

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
