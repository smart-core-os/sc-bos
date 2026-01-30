package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

// runScan executes the scan command
func runScan() {
	if *scanOrg == "" {
		log.Fatal("Organization name is required. Use -org flag.")
	}

	if *scanOutput == "" {
		log.Fatal("Output file is required. Use -output flag.")
	}

	githubToken := *scanToken
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" {
		log.Fatal("GitHub token is required. Use -token flag or set GITHUB_TOKEN environment variable.")
	}

	ctx := context.Background()

	if *verbose {
		log.Printf("Scanning imports from %s in organization %s", *scanModule, *scanOrg)
	}

	client := &http.Client{Timeout: 30 * time.Second}

	cache, err := newAPICache(*scanCacheDir)
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	if *verbose {
		log.Printf("Using cache directory: %s", cache.cacheDir)
	}

	repos, err := getOrgRepos(ctx, client, *scanOrg, githubToken, cache)
	if err != nil {
		log.Fatalf("Failed to get repositories: %v", err)
	}

	if *verbose {
		log.Printf("Found %d repositories (API calls: %d, cache hits: %d)",
			len(repos), cache.apiCalls, cache.cacheHits)
	}

	result := ScanResult{
		BaseModule:   *scanModule,
		Organization: *scanOrg,
		TotalRepos:   len(repos),
		PackageUsage: make(map[string][]ImportInfo),
		Timestamp:    time.Now(),
	}

	reposWithGoFiles := 0
	reposWithImports := 0

	baseRepoName := extractRepoNameFromModule(*scanModule)

	for _, repo := range repos {
		if baseRepoName != "" && repo.FullName == baseRepoName {
			if *verbose {
				log.Printf("Skipping %s - this is the base module repository", repo.FullName)
			}
			continue
		}

		if *verbose {
			log.Printf("Scanning repository: %s (language: %s, last push: %s)",
				repo.FullName, repo.Language, repo.PushedAt.Format("2006-01-02"))
		}

		imports, hasGoFiles, err := analyzeRepoImports(ctx, client, repo, githubToken, *scanModule, cache)
		if err != nil {
			if *verbose {
				log.Printf("Failed to scan %s: %v (API calls: %d, cache hits: %d)",
					repo.FullName, err, cache.apiCalls, cache.cacheHits)
			}
			continue
		}

		if hasGoFiles {
			reposWithGoFiles++
		}

		if len(imports) > 0 {
			reposWithImports++
			for pkg, files := range imports {
				result.PackageUsage[pkg] = append(result.PackageUsage[pkg], ImportInfo{
					Repository: repo.FullName,
					Package:    pkg,
					Files:      files,
					LastPush:   repo.PushedAt,
					LastUpdate: repo.UpdatedAt,
				})
			}

			if *verbose {
				log.Printf("Found %d packages in %s (API calls: %d, cache hits: %d)",
					len(imports), repo.FullName, cache.apiCalls, cache.cacheHits)
			}
		}
	}

	result.ReposWithGoFiles = reposWithGoFiles
	result.ReposWithImports = reposWithImports

	if *verbose {
		log.Printf("Scan complete. Total API calls: %d, cache hits: %d (%.1f%% cache hit rate)",
			cache.apiCalls, cache.cacheHits,
			float64(cache.cacheHits)/float64(cache.apiCalls+cache.cacheHits)*100)
	}

	if err := saveScanResult(result, *scanOutput); err != nil {
		log.Fatalf("Failed to save scan result: %v", err)
	}

	log.Printf("Scan results written to %s", *scanOutput)
}

var (
	// Scan subcommand flags
	scanCmd      = flag.NewFlagSet("scan", flag.ExitOnError)
	scanOrg      = scanCmd.String("org", "", "GitHub organization name to scan")
	scanToken    = scanCmd.String("token", "", "GitHub personal access token (or set GITHUB_TOKEN env var)")
	scanModule   = scanCmd.String("base-module", "github.com/smart-core-os/sc-bos", "Base module to look for imports")
	scanOutput   = scanCmd.String("output", "", "Output file path for scan results (required)")
	scanCacheDir = scanCmd.String("cache-dir", "", "Cache directory for API responses (default: ~/.package-import-analyzer-cache)")
)
