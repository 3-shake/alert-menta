package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	// コマンドライン引数を解析する
	var (
		repo        = flag.String("repo", "", "Repository name")
		owner       = flag.String("owner", "", "Repository owner")
		issueNumber = flag.Int("issue", 0, "Issue number")
		// commentBody = flag.String("comment", "", "Comment body")
		token = flag.String("token", "", "GitHub token")
	)
	flag.Parse()

	if *repo == "" || *owner == "" || *issueNumber == 0 || *token == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// OAuth2トークンを使用してGitHubクライアントを作成
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// PRにコメントを投稿
	issue, _, _ := client.Issues.Get(ctx, *owner, *repo, *issueNumber)
	fmt.Println("Title:", issue.GetTitle())
	fmt.Println("Body:", issue.GetBody())
	fmt.Println("A number of comments:", issue.GetComments())
	// comment, _, err := client.Issues.GetComment(ctx, *owner, *repo, 1)
	// fmt.Println(comment, err)
}
