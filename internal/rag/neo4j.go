package rag

import (
	"bytes"
	"context"
	"fmt"
	"strings"
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
	contentProp   string
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
		contentProp:   "text",
	}, nil
}

func (r *Neo4jRetriever) Close() {
	r.session.Close(r.context)
	r.driver.Close(r.context)
}

func (r *Neo4jRetriever) TestConnection() error {
	result, err := r.session.Run(r.context, "MATCH (n) RETURN n LIMIT 1", nil)
	if err != nil {
		return err
	}
	for result.Next(r.context) {
		record := result.Record()
		// レコードの各フィールドを処理します。
		for i, value := range record.Values {
			fmt.Printf("%s: %v\n", record.Keys[i], value)
		}
		fmt.Println("---")
	}
	fmt.Println(result)
	return nil
}

func (r *Neo4jRetriever) Retrieve(emb ai.EmbeddingModel, query string, options Options) ([]Document, error) {
	// var documents []Document
	embedding, err := emb.GetEmbedding(query)
	if err != nil {
		return nil, err
	}
	results, err := r.retrieveHybrid(embedding, query, options)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *Neo4jRetriever) retrieveHybrid(embedding []float32, query string, options Options) ([]Document, error) {
	// []float32 to string
	embeddingStr := fmt.Sprintf("%v", embedding)
	embeddingStr = strings.ReplaceAll(embeddingStr, " ", ", ")
	cypher, err := r.HybridSearch(embeddingStr, query)
	if err != nil {
		fmt.Errorf("Error: %v", err)
		return nil, err
	}

	documents, err := r.runCypher(cypher, nil)
	if err != nil {
		fmt.Errorf("Error: %v", err)
		return nil, err
	}
	return documents, nil
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
        RETURN node.source AS id, node.{{.Content}} AS text, score`)
	type Cypher struct {
		K         int
		Embedding string
		Query     string
		Content   string
	}
	var buf bytes.Buffer
	err := cypherTemplate.Execute(&buf, Cypher{5, embedding, r.sanitizeQuery(query), r.contentProp})
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	cypher := buf.String()
	fmt.Println(cypher)
	return cypher, nil
}

func (r *Neo4jRetriever) runCypher(query string, params map[string]interface{}) ([]Document, error) {
	var documents []Document
	result, err := r.session.Run(r.context, query, params)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	for result.Next(r.context) {
		record := result.Record()
		// レコードの各フィールドを処理します。
		for i, value := range record.Values {
			fmt.Printf("%s: %v\n", record.Keys[i], value)
		}
		fmt.Println("---")
		id, _, _ := neo4j.GetRecordValue[string](record, "id")
		content, _, _ := neo4j.GetRecordValue[string](record, "text")
		score, _, _ := neo4j.GetRecordValue[float64](record, "score")
		documents = append(documents, Document{
			Id:      id,
			Content: content,
			Score:   score,
		})
	}
	fmt.Println(len(documents))
	if len(documents) == 0 {
		fmt.Println("Error:", "No results found")
		return nil, fmt.Errorf("No results found")
	}
	return documents, nil
}

func (r *Neo4jRetriever) sanitizeQuery(query string) string {
	// Escaping special characters so that they can be interpreted as cypher
	newQuery := strings.ReplaceAll(query, "\"", "\\\"")
	newQuery = strings.ReplaceAll(newQuery, "'", "\\'")
	newQuery = strings.ReplaceAll(newQuery, "\n", "\\n")
	newQuery = strings.ReplaceAll(newQuery, "\r", "\\r")
	newQuery = strings.ReplaceAll(newQuery, "$", "\\$")
	newQuery = strings.ReplaceAll(newQuery, ":", "\\:")
	newQuery = strings.ReplaceAll(newQuery, "/", "\\/")
	newQuery = strings.ReplaceAll(newQuery, "[", "\\[")
	newQuery = strings.ReplaceAll(newQuery, "]", "\\]")
	newQuery = strings.ReplaceAll(newQuery, "(", "\\(")
	newQuery = strings.ReplaceAll(newQuery, ")", "\\)")
	newQuery = strings.ReplaceAll(newQuery, "{", "\\{")
	newQuery = strings.ReplaceAll(newQuery, "}", "\\}")
	newQuery = strings.ReplaceAll(newQuery, "~", "\\~")
	newQuery = strings.ReplaceAll(newQuery, "^", "\\^")
	return newQuery
}
