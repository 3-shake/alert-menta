package utils

import (
	"testing"
)

func TestCloneAndPullAllBranches(t *testing.T) {
	// テスト用のリモートリポジトリURL (公開されている小さなリポジトリ)
	testRepoURL := "https://github.com/3-shake/alert-menta.git"

	// 関数を呼び出す
	repo, err := CloneRepository(testRepoURL, nil)
	if err != nil {
		t.Fatalf("CloneAndPullAllBranches failed: %v", err)
	}

	branches, err := GetBranches(repo, []string{})
	if err != nil {
		t.Fatalf("GetAllBranches failed: %v", err)
	}
	t.Logf("GetAllBranches: %v", branches)

	if err := SwitchBranch(repo, "develop"); err != nil {
		t.Fatalf("SwitchBranch failed: %v", err)
	}

	files, err := ListFiles(repo)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}
	t.Logf("ListFiles: %v", files)

	for _, file := range files {
		if file.Path == ".gitignore" {
			content, err := GetFileContent(repo, file)
			if err != nil {
				t.Fatalf("GetFileContent failed: %v", err)
			}
			t.Logf("GetFileContent: %v: %v", file.Path, content)
		}
	}
}
