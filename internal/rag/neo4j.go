package rag

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jRetriever struct {
	session       neo4j.SessionWithContext
	driver        neo4j.DriverWithContext
	context       context.Context
	fulltextIndex string
	vectorIndex   string
}

func NewNeo4jRetriever(uri, username, password, fulltextIndex, vectorIndex string) (*Neo4jRetriever, error) {
	// driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, err
	}
	// session, err := driver.Session(neo4j.AccessModeRead)
	ctx := context.Background()
	session := driver.NewSession(ctx,
		neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	if err != nil {
		return nil, err
	}
	return &Neo4jRetriever{
		session:       session,
		driver:        driver,
		context:       ctx,
		fulltextIndex: fulltextIndex,
		vectorIndex:   vectorIndex,
	}, nil
}

func (r *Neo4jRetriever) Close() {
	r.session.Close(r.context)
	r.driver.Close(r.context)
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

func (r *Neo4jRetriever) StructuredRetriever(question string) (string, error) {
	query := fmt.Sprintf("MATCH (n:Question) WHERE n.text = '%s' RETURN n", question)
	_, err := r.session.Run(r.context, query, nil)
	if err != nil {
		return "", err
	}
	// return result.Single().GetByIndex(0).(neo4j.Node).Props()["answer"].(string), nil
	return "", nil
}

func (r *Neo4jRetriever) UnstructuredRetriever(question string) (string, error) {
	// query := fmt.Sprintf("MATCH (n:Question) WHERE n.text = '%s' RETURN n", question)
	return "", nil
}

func (r *Neo4jRetriever) HybridSearch(embedding, query string) (string, error) {
	cypherTemplate, _ := template.New("cypher").Parse(`CALL {
                CALL db.index.vector.queryNodes("vector", {{.K}}, {{.Embedding}})
                YIELD node, score
                WITH collect({node:node, score:score}) AS nodes, max(score) AS max
                UNWIND nodes AS n
                RETURN n.node AS node, (n.score / max) AS score
                UNION
                CALL db.index.fulltext.queryNodes("keyword", "{{.Query}}", {limit: {{.K}}})
                YIELD node, score
                WITH collect({node:node, score:score}) AS nodes, max(score) AS max
                UNWIND nodes AS n
                RETURN n.node AS node, (n.score / max) AS score
        }
        WITH node, max(score) AS score ORDER BY score DESC LIMIT {{.K}}
        RETURN node.{{.Content}} AS text, score`)
	type Cypher struct {
		K         int
		Embedding string
		Query     string
		Content   string
	}
	var buf bytes.Buffer
	err := cypherTemplate.Execute(&buf, Cypher{5, embedding, query, "source"})
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	result := buf.String()
	fmt.Println(result)
	return result, nil
}

func (r *Neo4jRetriever) runCypher(query string, params map[string]interface{}) ([]Document, error) {
	var documents []Document
	result, err := r.session.Run(r.context, query, params)
	if err != nil {
		return nil, err
	}
	for result.Next(r.context) {
		record := result.Record()
		documents = append(documents, Document{
			Id:      record.GetByIndex(0).(string),
			Content: record.GetByIndex(1).(string),
			Score:   record.GetByIndex(2).(float64),
		})
	}
	return documents, nil
}
