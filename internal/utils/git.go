package utils

import (
	"fmt"
	// "io/ioutil"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	// "github.com/go-git/go-git/v5/storage/memory"
)

// Path 構造体: ファイルパスとブランチ名を保持
type Path struct {
	FilePath string
	Branch   string
}

func GetAllFilesFromAllBranches(repoPath string, ignoreList, branchList []string) ([]Path, error) {

	var allPaths []Path

	if ignoreList == nil {
		ignoreList = []string{}
	}
	if branchList == nil {
		branchList = []string{}
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	ignoreMap := make(map[string]bool)
	for _, item := range ignoreList {
		ignoreMap[item] = true
	}

	branchRefs, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	err = branchRefs.ForEach(func(branchRef *plumbing.Reference) error {
		branchName := branchRef.Name().Short()

		if !slices.Contains(branchList, branchName) {
			return nil
		}

		commit, err := repo.CommitObject(branchRef.Hash())
		if err != nil {
			return fmt.Errorf("failed to get commit for branch %s: %w", branchName, err)
		}

		tree, err := commit.Tree()
		if err != nil {
			return fmt.Errorf("failed to get tree for branch %s: %w", branchName, err)
		}

		tree.Files().ForEach(func(file *object.File) error {

			baseName := filepath.Base(file.Name)
			if _, ok := ignoreMap[baseName]; ok {
				return nil
			}

			for _, ignoreItem := range ignoreList {
				matched, _ := filepath.Match(ignoreItem, file.Name)
				if matched {
					return nil
				}
			}

			allPaths = append(allPaths, Path{
				FilePath: file.Name,
				Branch:   branchName,
			})
			return nil
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return allPaths, nil
}

func Tree(path string, level int) error {
	// prefix を計算 (インデント用)
	prefix := strings.Repeat("│   ", level)

	// ファイル/ディレクトリ情報を取得
	files, err := os.ReadDir(path) // os.ReadDir がより新しいGoのバージョンでは推奨
	// files, err := ioutil.ReadDir(path) // os.ReadDir がより新しいGoのバージョンでは推奨
	if err != nil {
		return err
	}

	// 各エントリを処理
	for i, file := range files {
		isLast := i == len(files)-1 // 最後のエントリかどうかを判定

		// コネクタの文字列を決定
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		// ファイル/ディレクトリ名を表示
		fmt.Printf("%s%s%s\n", prefix, connector, file.Name())

		// ディレクトリの場合は再帰的に tree を呼び出す
		if file.IsDir() {
			newPath := filepath.Join(path, file.Name())
			if err := Tree(newPath, level+1); err != nil {
				return err
			}
		} else { // ファイルの場合は内容を出力
			filePath := filepath.Join(path, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("%s│   %sError reading file: %v\n", prefix, strings.Repeat("    ", level), err)
				continue //エラーでも続行
			}

			// 内容をprefix付きで出力
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				fmt.Printf("%s│   %s\n", prefix, line)
			}
		}
	}
	return nil
}

func CollectFiles(rootPath string, ignoreList []string) ([]string, error) {
	var filePaths []string

	if ignoreList == nil {
		ignoreList = []string{}
	}

	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("error converting to absolute path: %w", err)
	}

	// transform ignoreList to map for faster search
	ignoreMap := make(map[string]bool)
	for _, item := range ignoreList {
		ignoreMap[item] = true
	}

	err = filepath.WalkDir(absRootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if _, ok := ignoreMap[d.Name()]; ok {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(absRootPath, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}

		filePaths = append(filePaths, relPath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return filePaths, nil
}
