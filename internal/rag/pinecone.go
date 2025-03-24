package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/utils"
	"github.com/go-git/go-git/v5"
	"github.com/pinecone-io/go-pinecone/v3/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

func prettifyStruct(obj interface{}) string {
	bytes, _ := json.MarshalIndent(obj, "", "  ")
	return string(bytes)
}

type PineconeClient struct {
	context   context.Context
	pc        *pinecone.Client
	indexName string
}

func GetPineconeIndexName(owner, repo string) string {
	indexName := owner + "-" + strings.ToLower(repo)
	indexName = strings.ReplaceAll(indexName, "_", "-")
	return indexName
}

func NewPineconeClient(indexName, apiKey string) (*PineconeClient, error) {
	ctx := context.Background()

	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		// ApiKey: os.Getenv("PINECONE_API_KEY"),
		ApiKey: apiKey,
	})

	if err != nil {
		// log.Fatalf("Failed to create Client: %v", err)
		return nil, fmt.Errorf("Failed to create Client: %v", err)
	}
	pcClient := &PineconeClient{context: ctx, pc: pc, indexName: indexName}
	err = pcClient.createIndex()
	if err != nil {
		// log.Fatalf("Failed to create index \"%v\": %v", indexName, err)
		return nil, fmt.Errorf("Failed to create index \"%v\": %v", indexName, err)
	}
	return pcClient, nil
}

func ConvertBranchtoDocuments(owner, repoName string, repo *git.Repository, branch utils.Branch) (*[]Document, error) {
	var docs []Document
	if err := utils.SwitchBranch(repo, branch.Name); err != nil {
		fmt.Printf("Failed to switch branch: %v\n", err)
	}
	for _, file := range branch.Files {
		content, err := utils.GetFileContent(repo, file)

		if err != nil && len(content) == 0 {
			continue
		}
		if err != nil {
			fmt.Printf("Failed to get file content: %s, %v\n", content, err)
			return nil, fmt.Errorf("Failed to get file content: %s@%s", branch.Name, file.Path)
		}

		// content のトークン数が 8192 以上の場合は分割する（とりあえずトークン数だけ切り取る）
		n, err := ai.NumberofTokens(content)
		if err != nil {
			return nil, fmt.Errorf("Failed to get number of tokens: %v", err)
		}
		if n > 8192 {
			content = content[:8192]
		}

		docs = append(docs, Document{
			Id:      branch.Name + "@" + file.Path,
			Content: content,
			Branch:  branch.Name,
			URL:     fmt.Sprintf("https://github.com/%v/%v/blob/%v/%v", owner, repoName, branch.Name, file.Path),
			Score:   0,
		})
	}

	return &docs, nil
}

func ConvertPathtoDocument(owner, repo string, path utils.Path, root string) (*Document, error) {
	contentBytes, err := os.ReadFile(filepath.Join(root, path.FilePath))
	if err != nil {
		log.Fatalf("Failed to read file \"%v\": %v", path.FilePath, err)
		return nil, err
	}
	content := string(contentBytes)

	if len(content) == 0 {
		// log.Fatalf("File \"%v\" is empty", path)
		return nil, fmt.Errorf("File \"%v\" is empty", path)
	}

	return &Document{
		Id:      path.Branch + "@" + path.FilePath,
		Content: content,
		Branch:  path.Branch,
		URL:     fmt.Sprintf("https://github.com/%v/%v/blob/%v/%v", owner, repo, path.Branch, path.FilePath),
		Score:   0,
	}, nil
}

func (pc *PineconeClient) createIndex() error {
	_, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err == nil {
		return nil
	}
	metric := pinecone.Cosine
	dimension := int32(1536)

	_, err = pc.pc.CreateServerlessIndex(pc.context, &pinecone.CreateServerlessIndexRequest{
		Name:      pc.indexName,
		Cloud:     pinecone.Aws,
		Region:    "us-east-1",
		Metric:    &metric,
		Dimension: &dimension,
		Tags:      &pinecone.IndexTags{"environment": "development"},
	})
	if err != nil {
		return err
	}
	return nil
}

func (pc *PineconeClient) Retrieve(query string, embedding ai.EmbeddingModel, options Options) ([]Document, error) {
	emb, err := embedding.GetEmbedding(query)
	if err != nil {
		return nil, err
	}
	return pc.RetrieveByVector(emb, options)
}

func (pc *PineconeClient) RetrieveByVector(vector []float32, options Options) ([]Document, error) {
	var docs []Document
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}

	if state, err := pc.waitUntilIndexReady(); !state {
		return nil, err
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "codebase"})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}

	topK := options.topK
	if topK == 0 {
		topK = 3 // Default topK value
	}

	res, err := idxConnection.QueryByVectorValues(pc.context, &pinecone.QueryByVectorValuesRequest{
		Vector:          vector,
		TopK:            topK,
		IncludeValues:   false,
		IncludeMetadata: true,
	})
	if err != nil {
		log.Fatalf("Error encountered when querying by vector: %v", err)
	} else {
		log.Printf(prettifyStruct(res))
	}
	for _, match := range res.Matches {
		doc := Document{
			Id:      match.Vector.Metadata.GetFields()["id"].GetStringValue(),
			Content: match.Vector.Metadata.GetFields()["content"].GetStringValue(),
			Branch:  match.Vector.Metadata.GetFields()["branch"].GetStringValue(),
			URL:     match.Vector.Metadata.GetFields()["url"].GetStringValue(),
			Score:   0,
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (pc *PineconeClient) QueryById(id string) (*Document, error) {
	var doc *Document
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}

	if state, err := pc.waitUntilIndexReady(); !state {
		return nil, err
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "codebase"})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}

	res, err := idxConnection.QueryByVectorId(pc.context, &pinecone.QueryByVectorIdRequest{
		VectorId:        id,
		TopK:            1,
		IncludeValues:   false,
		IncludeMetadata: true,
	})
	if err != nil {
		log.Fatalf("Error encountered when querying by vector: %v", err)
	} else {
		log.Printf(prettifyStruct(res))
	}

	tempDoc := res.Matches[0].Vector.Metadata.GetFields()
	doc = &Document{
		Id:      tempDoc["id"].GetStringValue(),
		Content: tempDoc["content"].GetStringValue(),
		Branch:  tempDoc["branch"].GetStringValue(),
		URL:     tempDoc["url"].GetStringValue(),
		Score:   0,
	}

	return doc, nil
}

func (pc *PineconeClient) convertIssueStructtoMap(issue Issue) map[string]interface{} {
	return map[string]interface{}{
		"id":      issue.Id,
		"content": issue.Content,
		"title":   issue.Title,
		"url":     issue.Url,
		"state":   issue.State,
	}
}

func (pc *PineconeClient) convertDocumentStructtoMap(doc Document) map[string]interface{} {
	return map[string]interface{}{
		"id":      doc.Id,
		"content": doc.Content,
		"branch":  doc.Branch,
		"url":     doc.URL,
		"score":   doc.Score,
	}
}

func (pc *PineconeClient) DeleteIndex() {
	err := pc.pc.DeleteIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to delete index \"%v\": %v", pc.indexName, err)
	}
}

func (pc *PineconeClient) DeleteRecords(ids []string) error {
	for _, id := range ids {
		err := pc.DeleteRecord(id)
		if err != nil {
			log.Fatalf("Failed to delete record with id %v: %v", id, err)
			return err
		}
	}
	return nil
}

func (pc *PineconeClient) DeleteRecord(id string) error {
	nameSpace := "codebase"
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
		return err
	}

	if state, err := pc.waitUntilIndexReady(); !state {
		return err
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: nameSpace})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
		return err
	}
	err = idxConnection.DeleteVectorsById(pc.context, []string{id})
	if err != nil {
		log.Fatalf("Failed to delete vectors: %v", err)
		return err
	} else {
		log.Printf("Successfully deleted vector with id %v!\n", id)
	}
	return nil
}

func (pc *PineconeClient) CreateCodebaseDB(docs []Document, embedding ai.EmbeddingModel) error {
	var vectors [][]float32
	for _, doc := range docs {
		// 1536 is the default embedding size for the Universal Sentence Encoder
		vector, err := embedding.GetEmbedding(doc.Content)
		if err != nil {
			return fmt.Errorf("Error getting embedding: %v", err) // MAX input length is 8192 in OpenAI
		}
		vectors = append(vectors, vector)
	}
	err := pc.UpsertWithStruct(docs, vectors)
	if err != nil {
		return fmt.Errorf("Error upserting vectors: %v", err)
	}
	return nil
}

func (pc *PineconeClient) UpsertWithStruct(docs []Document, vectors [][]float32) error {
	nameSpace := "codebase"
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}

	if state, err := pc.waitUntilIndexReady(); !state {
		return err
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: nameSpace})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}
	pcVectors := make([]*pinecone.Vector, len(docs))
	for i, doc := range docs {
		metadataMap := pc.convertDocumentStructtoMap(doc)
		metadata, err := structpb.NewStruct(metadataMap)
		if err != nil {
			log.Fatalf("Failed to create metadata map: %v", err)
		}
		pcVectors[i] = &pinecone.Vector{
			Id:       doc.Id,
			Values:   &vectors[i],
			Metadata: metadata,
		}
	}
	count, err := idxConnection.UpsertVectors(pc.context, pcVectors)
	if err != nil {
		log.Fatalf("Failed to upsert vectors: %v", err)
		return err
	} else {
		log.Printf("Successfully upserted %d vector(s)!\n", count)
	}
	return nil
}

// Referenced in https://github.com/pinecone-io/go-pinecone/blob/af29d07e7c68/pinecone/test_suite.go#L147
func (pc *PineconeClient) waitUntilIndexReady() (bool, error) {
	start := time.Now()
	delay := 5 * time.Second
	maxWaitTimeSeconds := 280 * time.Second

	for {
		index, err := pc.pc.DescribeIndex(pc.context, pc.indexName)

		if index.Status.Ready && index.Status.State == "Ready" {
			log.Printf("Index \"%s\" is ready after %f seconds\n", pc.indexName, time.Since(start).Seconds())
			return true, err
		}

		totalSeconds := time.Since(start)

		if totalSeconds >= maxWaitTimeSeconds {
			return false, fmt.Errorf("Index \"%s\" not ready after %f seconds", pc.indexName, totalSeconds.Seconds())
		}

		log.Printf("Index \"%s\" not ready yet, retrying... (%f/%f)\n", pc.indexName, totalSeconds.Seconds(), maxWaitTimeSeconds.Seconds())
		time.Sleep(delay)
	}
}
