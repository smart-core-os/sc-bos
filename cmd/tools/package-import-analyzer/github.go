package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// httpGetJSON makes an authenticated HTTP GET request to the GitHub API and decodes the JSON response
func httpGetJSON(ctx context.Context, client *http.Client, url, githubToken string, target any, cache *APICache) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+githubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Track API call
	cache.apiCalls++
	if *veryVerbose {
		log.Printf("API CALL to %s (total: %d)", url, cache.apiCalls)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// httpGetJSONWithCache checks cache first, then makes API call if needed
func httpGetJSONWithCache(ctx context.Context, client *http.Client, url, githubToken string, target any, cache *APICache, ttl time.Duration) error {
	// Try cache first
	if cache.get(url, target) {
		return nil // Cache hit
	}

	// Cache miss - make API call
	if err := httpGetJSON(ctx, client, url, githubToken, target, cache); err != nil {
		return err
	}

	// Cache the response
	if err := cache.set(url, target, ttl); err != nil && *veryVerbose {
		log.Printf("Failed to cache response for %s: %v", url, err)
	}

	return nil
}

// getOrgRepos retrieves all repositories for a GitHub organization
func getOrgRepos(ctx context.Context, client *http.Client, org, githubToken string, cache *APICache) ([]GitHubRepo, error) {
	var allRepos []GitHubRepo
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?page=%d&per_page=%d", org, page, perPage)

		var repos []GitHubRepo
		if err := httpGetJSONWithCache(ctx, client, url, githubToken, &repos, cache, 30*time.Minute); err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)

		if len(repos) < perPage {
			break
		}
		page++
	}

	return allRepos, nil
}

// repoHasGoFiles checks if a repository has Go files using the GitHub Languages API
func repoHasGoFiles(ctx context.Context, client *http.Client, repo GitHubRepo, githubToken string, cache *APICache) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/languages", repo.FullName)

	var languages map[string]int
	if err := httpGetJSONWithCache(ctx, client, url, githubToken, &languages, cache, 24*time.Hour); err != nil {
		if err.Error() == "not found" {
			return false, nil // Repository not accessible or empty
		}
		return false, err
	}

	// Check if "Go" is in the languages map
	_, hasGo := languages["Go"]
	return hasGo, nil
}

// getRepoTree uses the Git Trees API to get the full file tree of the repository
// This is much more efficient than recursively fetching directory contents
func getRepoTree(ctx context.Context, client *http.Client, repo GitHubRepo, githubToken string, cache *APICache) (*GitTree, error) {
	// Use the Git Trees API with recursive=1 to get all files in one request
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", repo.FullName, repo.DefaultBranch)

	var tree GitTree
	if err := httpGetJSONWithCache(ctx, client, url, githubToken, &tree, cache, 1*time.Hour); err != nil {
		if err.Error() == "not found" {
			return nil, nil // Empty repository or branch doesn't exist
		}
		return nil, err
	}

	if tree.Truncated {
		log.Printf("Warning: tree for %s was truncated, some files may be missing", repo.FullName)
	}

	return &tree, nil
}

// repoHasModuleDependency checks if any go.mod file in the repository contains the base module as a dependency
// Returns the list of module directories (directories containing go.mod with the dependency)
// This helps avoid scanning repositories that don't use the module at all
func repoHasModuleDependency(ctx context.Context, client *http.Client, repo GitHubRepo, githubToken, baseModule string, cache *APICache, tree *GitTree) ([]string, error) {
	// Find all go.mod files in the tree
	var goModPaths []string
	for _, item := range tree.Tree {
		if item.Type == "blob" && strings.HasSuffix(item.Path, "go.mod") {
			goModPaths = append(goModPaths, item.Path)
		}
	}

	// If no go.mod files found, return nil to scan all files (conservative approach)
	if len(goModPaths) == 0 {
		if *verbose {
			log.Printf("No go.mod files found in %s, will scan all Go files", repo.FullName)
		}
		return nil, nil
	}

	// Check each go.mod file for the dependency and collect matching module directories
	var moduleDirs []string
	for _, goModPath := range goModPaths {
		hasDep, err := checkGoModFile(ctx, client, repo, githubToken, baseModule, goModPath, cache)
		if err != nil {
			if *verbose {
				log.Printf("Failed to check %s/%s: %v", repo.FullName, goModPath, err)
			}
			continue // Try other go.mod files
		}
		if hasDep {
			// Extract the directory containing this go.mod
			moduleDir := strings.TrimSuffix(goModPath, "/go.mod")
			if moduleDir == "go.mod" {
				moduleDir = "" // Root directory
			}
			moduleDirs = append(moduleDirs, moduleDir)
			if *verbose {
				log.Printf("Found dependency on %s in %s/%s", baseModule, repo.FullName, goModPath)
			}
		}
	}

	return moduleDirs, nil
}

// checkGoModFile checks if a specific go.mod file contains the base module as a dependency
func checkGoModFile(ctx context.Context, client *http.Client, repo GitHubRepo, githubToken, baseModule, goModPath string, cache *APICache) (bool, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo.FullName, goModPath)

	var fileData map[string]any
	if err := httpGetJSONWithCache(ctx, client, url, githubToken, &fileData, cache, 6*time.Hour); err != nil {
		return false, err
	}

	content, ok := fileData["content"].(string)
	if !ok {
		return false, fmt.Errorf("no content in go.mod response")
	}

	// Decode base64 content
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
	if err != nil {
		return false, err
	}

	// Simple string search for the base module in go.mod
	// This is faster than parsing the entire go.mod file
	goModContent := string(decoded)
	return strings.Contains(goModContent, baseModule+" "), nil
}

// analyzeRepoImports scans a repository for imports of the base module
func analyzeRepoImports(ctx context.Context, client *http.Client, repo GitHubRepo, githubToken, baseModule string, cache *APICache) (map[string][]string, bool, error) {
	// First check if the repository has Go files using the Languages API
	hasGoFiles, err := repoHasGoFiles(ctx, client, repo, githubToken, cache)
	if err != nil {
		return nil, false, err
	}

	// If no Go files, skip scanning
	if !hasGoFiles {
		return make(map[string][]string), false, nil
	}

	// Get the Git Tree for the repository (we'll use this for both go.mod check and finding Go files)
	tree, err := getRepoTree(ctx, client, repo, githubToken, cache)
	if err != nil {
		return nil, hasGoFiles, err
	}

	if tree == nil {
		// Empty repository
		return make(map[string][]string), hasGoFiles, nil
	}

	// Check go.mod files to see if the base module is a dependency
	// This returns the list of module directories that have the dependency
	moduleDirs, err := repoHasModuleDependency(ctx, client, repo, githubToken, baseModule, cache, tree)
	if err != nil && *verbose {
		log.Printf("Failed to check go.mod for %s: %v (will scan all files anyway)", repo.FullName, err)
		// Continue anyway - repo might not have go.mod or it might be an error
		moduleDirs = nil // Scan all files
	} else if len(moduleDirs) == 0 && moduleDirs != nil {
		// We found go.mod files but none have the dependency
		if *verbose {
			log.Printf("Skipping %s - no dependency on %s in any go.mod", repo.FullName, baseModule)
		}
		return make(map[string][]string), hasGoFiles, nil
	}

	// Get all Go files from the tree, filtered by module directories
	goFiles := extractGoFilesFromTree(tree, moduleDirs)

	if *verbose && moduleDirs != nil {
		log.Printf("Analyzing %d Go files in %d module(s) for %s", len(goFiles), len(moduleDirs), repo.FullName)
	}

	imports := make(map[string][]string)

	// Analyze each Go file
	for _, filePath := range goFiles {
		if err := analyzeGoFile(ctx, client, repo, githubToken, baseModule, filePath, imports, cache); err != nil && *verbose {
			log.Printf("Failed to analyze Go file %s/%s: %v", repo.FullName, filePath, err)
		}
	}

	return imports, hasGoFiles, nil
}

// analyzeGoFile analyzes a single Go file for imports
func analyzeGoFile(ctx context.Context, client *http.Client, repo GitHubRepo, githubToken, baseModule, filePath string, imports map[string][]string, cache *APICache) error {
	// Get file contents using cache
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo.FullName, filePath)

	var fileData map[string]any
	if err := httpGetJSONWithCache(ctx, client, url, githubToken, &fileData, cache, 6*time.Hour); err != nil {
		return err
	}

	content, ok := fileData["content"].(string)
	if !ok {
		return fmt.Errorf("no content in file response")
	}

	// Decode base64 content
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
	if err != nil {
		return err
	}

	// Parse Go file and extract imports
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, decoded, parser.ImportsOnly)
	if err != nil {
		return err
	}

	for _, importSpec := range node.Imports {
		importPath := strings.Trim(importSpec.Path.Value, "\"")

		if strings.HasPrefix(importPath, baseModule+"/") || importPath == baseModule {
			// This is an import from our base module
			imports[importPath] = append(imports[importPath], filePath)
		}
	}

	return nil
}
