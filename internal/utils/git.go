package utils

import (
	"fmt"
	"log"
	// "reflect"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

type MemoryFile struct {
	Size int64
	Path string
}

type Path struct {
	FilePath string
	Branch   string
}

type Branch struct {
	Name  string
	Files []MemoryFile
}

type AuthOptions struct {
	Username string
	Token    string
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
	prefix := strings.Repeat("│   ", level)

	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for i, file := range files {
		isLast := i == len(files)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		fmt.Printf("%s%s%s\n", prefix, connector, file.Name())

		if file.IsDir() {
			newPath := filepath.Join(path, file.Name())
			if err := Tree(newPath, level+1); err != nil {
				return err
			}
		} else {
			filePath := filepath.Join(path, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("%s│   %sError reading file: %v\n", prefix, strings.Repeat("    ", level), err)
				continue
			}

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

func CloneRepository(url string, authOptions *AuthOptions) (*git.Repository, error) {
	fs := memfs.New()

	var auth *http.BasicAuth
	if authOptions == nil {
		authOptions = &AuthOptions{}
	} else {
		auth = &http.BasicAuth{
			Username: authOptions.Username,
			Password: authOptions.Token,
		}
	}

	repo, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
		Depth:    1,
		Auth:     auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	err = PullAllRemoteBranch(repo, "origin", authOptions)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func SwitchBranch(repo *git.Repository, branchName string) error {
	force := true
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Force:  force,
		Keep:   !force,
	})
	if err != nil {
		return fmt.Errorf("failed to switch branch: %w", err)
	}
	return nil
}

func PullAllRemoteBranch(repo *git.Repository, remoteName string, authOptions *AuthOptions) error {
	var auth *http.BasicAuth
	if authOptions == nil {
		authOptions = &AuthOptions{}
	} else {
		auth = &http.BasicAuth{
			Username: authOptions.Username,
			Password: authOptions.Token,
		}
	}

	remote, err := repo.Remote(remoteName)
	if err != nil {
		log.Fatalf("Failed to get remote 'origin': %v", err)
	}

	refs, err := remote.List(&git.ListOptions{Auth: auth})
	if err != nil {
		log.Fatalf("Failed to list remote refs: %v", err)
	}

	for _, ref := range refs {
		if ref.Name().IsBranch() {
			localBranchName := ref.Name().Short()
			remoteBranchName := ref.Name()
			remoteBranchRefSpec := fmt.Sprintf("+%s:%s", remoteBranchName, plumbing.NewBranchReferenceName(localBranchName))

			_, err := repo.Reference(plumbing.NewBranchReferenceName(localBranchName), true)
			if err != nil {
				headRef, err := repo.Head()
				if err != nil && err != plumbing.ErrReferenceNotFound {
					log.Printf("get head err=%v", err)
					continue
				}

				var commit *object.Commit
				if err == plumbing.ErrReferenceNotFound {
					commit, err = repo.CommitObject(ref.Hash())
					if err != nil {
						log.Printf("branch %s commit not found. err=%v\n", localBranchName, err)
						continue
					}
				} else {
					commit, err = repo.CommitObject(headRef.Hash())
					if err != nil {
						log.Printf("branch %s commit not found from haed ref. err=%v\n", localBranchName, err)
						continue
					}
				}

				newRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(localBranchName), commit.Hash)
				if err := repo.Storer.SetReference(newRef); err != nil {
					log.Printf("failed to create local branch %s: %v", localBranchName, err)
					continue
				}
				_, _ = repo.Reference(plumbing.NewBranchReferenceName(localBranchName), true)

			}
			err = repo.CreateBranch(&config.Branch{
				Name:   localBranchName,
				Remote: remoteName,
				Merge:  remoteBranchName,
			})

			if err != nil {
				log.Printf("Error setting up tracking for %s: %v\n", localBranchName, err)
				continue
			}

			err = repo.Fetch(&git.FetchOptions{
				RemoteName: remoteName,
				RefSpecs:   []config.RefSpec{config.RefSpec(remoteBranchRefSpec)},
				Progress:   os.Stdout,
				Force:      true,
				Auth:       auth,
			})

			if err != nil && err != git.NoErrAlreadyUpToDate {
				log.Printf("fetch error (branch %s): %v", localBranchName, err)
				continue
			}
			fmt.Printf("Branch %s set up to track remote branch %s from origin.\n", localBranchName, remoteBranchName)
		}
	}
	fmt.Println("All remote branches have been pulled to local branches.")
	return nil
}

func ListFiles(repo *git.Repository) ([]MemoryFile, error) {
	files, err := recursiveListFiles(repo, ".")
	return files, err
}

func recursiveListFiles(repo *git.Repository, path string) ([]MemoryFile, error) {
	// var files []string
	files := []MemoryFile{}
	wt, err := repo.Worktree()
	if err != nil {
		return files, err
	}

	wtfiles, err := wt.Filesystem.ReadDir(path)
	if err != nil {
		return files, err
	}
	for _, file := range wtfiles {
		if file.IsDir() {
			// var tempFiles []string
			tempFiles := []MemoryFile{}
			if path == "." {
				tempFiles, err = recursiveListFiles(repo, file.Name())
			} else {
				tempFiles, err = recursiveListFiles(repo, path+"/"+file.Name())
			}
			if err != nil {
				return files, err
			}
			files = append(files, tempFiles...)
		} else {
			if path == "." {
				files = append(files, MemoryFile{Path: file.Name(), Size: file.Size()})
			} else {
				files = append(files, MemoryFile{Path: path + "/" + file.Name(), Size: file.Size()})
			}
		}
	}
	return files, err
}

func GetFileContent(repo *git.Repository, file MemoryFile) (string, error) {
	byteContent := make([]byte, file.Size)
	wt, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	billyFile, err := wt.Filesystem.Open(file.Path)
	if err != nil {
		return "", err
	}

	_, err = billyFile.Read(byteContent)
	if err != nil {
		return "", err
	}

	return string(byteContent), err
}

func GetAllBranchNames(repo *git.Repository) ([]string, error) {
	branchRefs, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}
	var branches []string
	err = branchRefs.ForEach(func(branchRef *plumbing.Reference) error {
		branches = append(branches, branchRef.Name().Short())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}
	return branches, nil
}

// if specifiedBranch is empty, get all branches
func GetBranches(repo *git.Repository, specifiedBranch []string) ([]Branch, error) {
	var branches []Branch
	branchNames, err := GetAllBranchNames(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch names: %w", err)
	}

	for _, branchName := range branchNames {
		if len(specifiedBranch) > 0 && !slices.Contains(specifiedBranch, branchName) {
			continue
		}

		err := SwitchBranch(repo, branchName)
		if err != nil {
			return nil, fmt.Errorf("failed to switch branch: %w", err)
		}

		memFiles, err := ListFiles(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		branches = append(branches, Branch{
			Name:  branchName,
			Files: memFiles,
		})
	}

	return branches, nil
}
