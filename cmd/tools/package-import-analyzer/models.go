package main

import "time"

// CacheEntry holds cached API response with metadata
type CacheEntry struct {
	Data      any       `json:"data"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GitHubRepo represents a GitHub repository
type GitHubRepo struct {
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	CloneURL      string    `json:"clone_url"`
	Language      string    `json:"language"`
	DefaultBranch string    `json:"default_branch"`
	PushedAt      time.Time `json:"pushed_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GitTreeItem represents an item in a Git tree
type GitTreeItem struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

// GitTree represents a Git tree structure
type GitTree struct {
	SHA       string        `json:"sha"`
	URL       string        `json:"url"`
	Tree      []GitTreeItem `json:"tree"`
	Truncated bool          `json:"truncated"`
}

// ImportInfo contains information about a package import
type ImportInfo struct {
	Repository string    `json:"repository"`
	Package    string    `json:"package"`
	Files      []string  `json:"files"`
	LastPush   time.Time `json:"last_push"`
	LastUpdate time.Time `json:"last_update"`
}

// ScanResult contains the results of scanning repositories
type ScanResult struct {
	BaseModule       string                  `json:"base_module"`
	Organization     string                  `json:"organization"`
	TotalRepos       int                     `json:"total_repos"`
	ReposWithGoFiles int                     `json:"repos_with_go_files"`
	ReposWithImports int                     `json:"repos_with_imports"`
	PackageUsage     map[string][]ImportInfo `json:"package_usage"`
	Timestamp        time.Time               `json:"timestamp"`
}

// PackageTypeAnalysis contains analysis results grouped by package type
type PackageTypeAnalysis struct {
	PackageType    string         `json:"package_type"`
	Count          int            `json:"count"`
	Packages       []string       `json:"packages"`
	PackageCounts  map[string]int `json:"package_counts,omitempty"` // Import count for each package
	DependentRepos int            `json:"dependent_repos"`
	ExampleFiles   []string       `json:"example_files,omitempty"` // Sample files showing where imports occur
}

// RepoDependencyAnalysis contains analysis of repository dependencies
type RepoDependencyAnalysis struct {
	Repository     string `json:"repository"`
	PackageCount   int    `json:"package_count"`
	ImportingRepos int    `json:"importing_repos"`
}
