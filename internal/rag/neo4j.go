package rag

import (
	"context"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Neo4jRetriever struct {
	session       *neo4j.Session
	driver        neo4j.Driver
	fulltextIndex string
	vectorIndex   string
}

func NewNeo4jRetriever(uri, username, password, fulltextIndex, vectorIndex string) (*Neo4jRetriever, error) {
	driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, err
	}
	session, err := driver.Session(neo4j.AccessModeRead)
	if err != nil {
		return nil, err
	}
	return &Neo4jRetriever{
		session:       session,
		driver:        driver,
		fulltextIndex: fulltextIndex,
		vectorIndex:   vectorIndex,
	}, nil
}

func (r *Neo4jRetriever) Close() {
	r.session.Close()
	r.driver.Close()
}

func (r *Neo4jRetriever) Retrieve(emb ai.EmbeddingModel, query string, options Options) ([]Document, error) {
	// var documents []Document
	embedding, err := emb.GetEmbedding(query)
	if err != nil {
		return nil, err
	}
	results, err := r.retrieveHybrid(embedding, query, options)

	return results, nil
}

func (r *Neo4jRetriever) retrieveHybrid(embedding []float32, query string, options Options) ([]Document, error) {
	return nil, nil
}

func (r *Neo4jRetriever) retrieveFulltext(query string, options Options) ([]Document, error) {
	return nil, nil
}

func (r *Neo4jRetriever) runCypher(query string, params map[string]interface{}) ([]Document, error) {
	var documents []Document
	result, err := r.session.Run(query, params)
	if err != nil {
		return nil, err
	}
	for result.Next() {
		record := result.Record()
		documents = append(documents, Document{
			Id:      record.GetByIndex(0).(string),
			Content: record.GetByIndex(1).(string),
			Score:   record.GetByIndex(2).(float64),
		})
	}
	return documents, nil
}
