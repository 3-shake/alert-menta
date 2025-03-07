package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/utils"
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

func NewPineconeClient(indexName string) *PineconeClient {
	ctx := context.Background()

	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: os.Getenv("PINECONE_API_KEY"),
	})

	if err != nil {
		log.Fatalf("Failed to create Client: %v", err)
	}
	return &PineconeClient{context: ctx, pc: pc, indexName: indexName}
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

// func (pc *PineconeClient) Retrieve(embedding ai.EmbeddingModel, options Options) ([]Document, error) {
func (pc *PineconeClient) RetrieveByVector(vector []float32, options Options) ([]Document, error) {
	var docs []Document
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}
	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "codebase"})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}
	res, err := idxConnection.QueryByVectorValues(pc.context, &pinecone.QueryByVectorValuesRequest{
		Vector:          vector,
		TopK:            3,
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
		// vector := make([]float32, 1536)
		// var err error
		if err != nil {
			log.Fatalf("Error getting embedding: %v", err)
		}
		vectors = append(vectors, vector)
	}
	err := pc.UpsertWithStruct(docs, vectors)
	if err != nil {
		log.Fatalf("Error upserting docs: %v", err)
		return err
	}
	return nil
}

func (pc *PineconeClient) UpsertWithStruct(docs []Document, vectors [][]float32) error {
	nameSpace := "codebase"
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
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
