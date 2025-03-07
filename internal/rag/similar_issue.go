package rag

import (
	"fmt"
	"log"

	// "github.com/joho/godotenv"
	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	// gogithub "github.com/google/go-github/github"
	"github.com/pinecone-io/go-pinecone/v3/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

type Issue struct {
	Id      string
	Url     string
	Content string
	Title   string
	State   string
	// Source  string
}

func (pc *PineconeClient) TestUpsert(metadataMap map[string]interface{}, vector []float32) {
	indexName := "similar-issues"
	// Add to the main function:

	idxModel, err := pc.pc.DescribeIndex(pc.context, indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", indexName, err)
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "issues"})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}
	metadata, err := structpb.NewStruct(metadataMap)
	if err != nil {
		log.Fatalf("Failed to create metadata map: %v", err)
	}
	pcVector := []*pinecone.Vector{
		{
			Id:       "vec2",
			Values:   &vector,
			Metadata: metadata,
		},
	}

	count, err := idxConnection.UpsertVectors(pc.context, pcVector)
	if err != nil {
		log.Fatalf("Failed to upsert vectors: %v", err)
	} else {
		log.Printf("Successfully upserted %d vector(s)!\n", count)
	}
}

func (pc *PineconeClient) RetrieveIssue(vector []float32) string {
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}
	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "issues"})
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
	text := "## Other issues similar to this one are: \n"
	text += fmt.Sprintf("1. [%s #%s (%s)](%s)\n", res.Matches[0].Vector.Metadata.GetFields()["title"].GetStringValue(), res.Matches[0].Vector.Metadata.GetFields()["id"].GetStringValue(), res.Matches[0].Vector.Metadata.GetFields()["state"].GetStringValue(), res.Matches[0].Vector.Metadata.GetFields()["url"].GetStringValue())
	text += fmt.Sprintf("1. [%s #%s (%s)](%s)\n", res.Matches[1].Vector.Metadata.GetFields()["title"].GetStringValue(), res.Matches[1].Vector.Metadata.GetFields()["id"].GetStringValue(), res.Matches[1].Vector.Metadata.GetFields()["state"].GetStringValue(), res.Matches[1].Vector.Metadata.GetFields()["url"].GetStringValue())
	text += fmt.Sprintf("3. [%s #%s (%s)](%s)\n", res.Matches[2].Vector.Metadata.GetFields()["title"].GetStringValue(), res.Matches[2].Vector.Metadata.GetFields()["id"].GetStringValue(), res.Matches[2].Vector.Metadata.GetFields()["state"].GetStringValue(), res.Matches[2].Vector.Metadata.GetFields()["url"].GetStringValue())
	return text
}

func (pc *PineconeClient) UpsertIssuesWithStruct(issues []Issue, vectors [][]float32) error {
	nameSpace := "issues"
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}
	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: nameSpace})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}
	pcVectors := make([]*pinecone.Vector, len(issues))
	for i, issue := range issues {
		metadataMap := pc.convertIssueStructtoMap(issue)
		metadata, err := structpb.NewStruct(metadataMap)
		if err != nil {
			log.Fatalf("Failed to create metadata map: %v", err)
		}
		pcVectors[i] = &pinecone.Vector{
			Id:       issue.Id,
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

func (pc *PineconeClient) UpsertIssue(id string, metadataMap map[string]interface{}, vector []float32) error {
	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "issues"})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}
	metadata, err := structpb.NewStruct(metadataMap)
	if err != nil {
		log.Fatalf("Failed to create metadata map: %v", err)
	}
	pcVector := []*pinecone.Vector{
		{
			Id:       id,
			Values:   &vector,
			Metadata: metadata,
		},
	}

	count, err := idxConnection.UpsertVectors(pc.context, pcVector)
	if err != nil {
		log.Fatalf("Failed to upsert vectors: %v", err)
		return err
	} else {
		log.Printf("Successfully upserted %d vector(s)!\n", count)
	}
	return nil
}

func (pc *PineconeClient) UpsertIssueWithStruct(issue Issue, vector []float32) error {
	metadataMap := pc.convertIssueStructtoMap(issue)
	pc.UpsertIssue(issue.Id, metadataMap, vector)
	return nil
}

// Query the index
// func (pc *PineconeClient) GetSpecifiedData(id string) Issue {
func (pc *PineconeClient) GetSpecifiedData(id string) {
	// Add to the main function:

	idxModel, err := pc.pc.DescribeIndex(pc.context, pc.indexName)
	if err != nil {
		log.Fatalf("Failed to describe index \"%v\": %v", pc.indexName, err)
	}

	idxConnection, err := pc.pc.Index(pinecone.NewIndexConnParams{Host: idxModel.Host, Namespace: "issues"})
	if err != nil {
		log.Fatalf("Failed to create IndexConnection1 for Host %v: %v", idxModel.Host, err)
	}

	// metadataFilter, err := structpb.NewStruct(metadataMap)
	// if err != nil {
	// log.Fatalf("Failed to create metadata map: %v", err)
	// }

	res, err := idxConnection.QueryByVectorId(pc.context, &pinecone.QueryByVectorIdRequest{
		VectorId:        id,
		TopK:            1,
		IncludeValues:   true,
		IncludeMetadata: true,
	})

	if err != nil {
		log.Fatalf("Error encountered when querying by vector: %v", err)
	} else {
		log.Printf(prettifyStruct(res))
	}
	log.Println(res.Matches[0].Vector.Metadata.GetFields()["question"].GetStringValue())
	// return Issue{id: res.Matches["vector"][0]["metadata"], content: res.Matches["vector"][0]["content"], source: res.Matches["vector"][0]["source"]}
}

func (pc *PineconeClient) CreateIssueDB(issues []*github.GitHubIssue, embedding ai.EmbeddingModel) error {
	// github.GetAllIssues("pacificbelt30", "actios_tester", os.Getenv("GITHUB_TOKEN"))
	structIssues := make([]Issue, len(issues))
	var vectors [][]float32
	for i, issue := range issues {
		gissue, _ := issue.GetIssue()
		body, _ := issue.GetBody()
		if body == nil {
			body = new(string)
		}

		comments, _ := issue.GetComments()
		content := *body + "\n" + "Comments: "
		for _, comment := range comments {
			content += *comment.User.Login + ":" + *comment.Body + "\n"
		}

		structIssues[i] = Issue{
			Id:      fmt.Sprintf("%d", *gissue.Number),
			Url:     *gissue.HTMLURL,
			Content: content,
			Title:   *gissue.Title,
			State:   *gissue.State,
		}
		vector, err := embedding.GetEmbedding("Title:" + *gissue.Title + "Body:" + content)
		if err != nil {
			log.Fatalf("Error getting embedding: %v", err)
		}
		vectors = append(vectors, vector)
	}
	err := pc.UpsertIssuesWithStruct(structIssues, vectors)
	if err != nil {
		log.Fatalf("Error upserting issues: %v", err)
		return err
	}
	return nil
}
