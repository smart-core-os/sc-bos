# Organization Migration Tool

A command-line tool to migrate code references from the `vanti-dev` organization to the `smart-core-os` organization.

## Overview

Multiple repositories are being transferred from `vanti-dev` to `smart-core-os` on GitHub.
This tool helps update references in your codebase to match the new organization.

**Supported projects by default:**

- `sc-bos` - Smart Core Building Operating System
- `gobacnet` - Go BACnet library

You can also migrate custom projects using the `--project` flag.

## How It Works

The tool scans files in your project and applies specific replacements based on the preset you choose and the projects
you specify:

### What Gets Replaced

The tool handles references for any project specified with `--project` (default: `sc-bos,gobacnet`):

| Replacement Type     | Pattern                                  | Example                                          |
|----------------------|------------------------------------------|--------------------------------------------------|
| **Go imports**       | `github.com/vanti-dev/<project>`         | `import "github.com/vanti-dev/gobacnet"`         |
| **Protocol Buffers** | `github.com/vanti-dev/<project>`         | `option go_package = "github.com/vanti-dev/..."` |
| **GitHub workflows** | `github.com/vanti-dev/<project>`         | Go package paths in `.github/workflows/*.yml`    |
| **npm packages**     | `@vanti-dev/sc-bos` (sc-bos only)        | `"@vanti-dev/sc-bos": "^1.0.0"`                  |
| **Docker images**    | `ghcr.io/vanti-dev/<project>`            | `image: ghcr.io/vanti-dev/sc-bos:latest`         |
| **GitHub URLs**      | `https://github.com/vanti-dev/<project>` | `https://github.com/vanti-dev/gobacnet/...`      |
| **IDEA run configs** | `github.com/vanti-dev/<project>`         | Package paths in `.run/*.xml` files              |

### Presets

Presets control which replacements are applied:

| Preset   | What It Does                          | Use When                                      |
|----------|---------------------------------------|-----------------------------------------------|
| `go`     | Updates Go import paths               | Your project imports Go packages              |
| `js`     | Updates npm package references        | Your project uses npm packages                |
| `docker` | Updates Docker image references       | Your project uses Docker images               |
| `docs`   | Updates GitHub URLs                   | You want to update documentation links only   |
| `all`    | Applies all replacements (default)    | Your project uses multiple components         |

By default, all file types are scanned. Use `--type` to limit which files are processed.

## Installation

### Install from GitHub

```bash
# Install the latest version directly from GitHub
go install github.com/smart-core-os/sc-bos/cmd/tools/org-migration@main

# Then use it anywhere
org-migration --dry-run --preset go --path /path/to/your/project  # --path defaults to .
```

### Run from source

If you have the repository cloned locally:

```bash
go run ./cmd/tools/org-migration --dry-run --preset go --path /path/to/your/project
```

## Quick Start

```bash
# Preview what would change (always do this first!)
org-migration --dry-run --preset go --path /path/to/your/project

# Apply the changes
org-migration --preset go --path /path/to/your/project
```

## Usage Examples

```bash
# Basic usage - migrate both sc-bos and gobacnet with default settings
org-migration --dry-run --preset go --path ~/my-project

# Migrate specific project only
org-migration --preset go --project gobacnet --path ~/my-project

# Migrate custom project with specific branch
org-migration --preset go --project myproject@develop --path ~/my-project

# Override default branch for known projects
org-migration --preset go --project sc-bos@feature-branch --path ~/my-project

# Multiple presets (comma-separated or repeated flags)
org-migration --preset go,docker --path ~/my-project
org-migration --preset go --preset docker --path ~/my-project

# Scan specific file types only
org-migration --type go,mod --preset go --path ~/my-project

# Verbose output to see all changes
org-migration --dry-run --verbose --preset all --path .
```

**Note:** If you're running from source instead of using `go install`, replace `org-migration` with
`go run ./cmd/tools/org-migration` in the examples above.

## Command-Line Options

```
--dry-run       Preview changes without modifying files (recommended first step)
--verbose       Show detailed output including line-by-line changes
--path PATH     Directory to scan (default: current directory)
--preset NAME   Which replacements to apply: all, go, js, docker, docs (default: all)
                Can be comma-separated (--preset go,docker) or repeated (--preset go --preset docker)
--type EXTS     File extensions to scan (default: all supported types)
                Can be comma-separated (--type go,mod) or repeated (--type go --type mod)
--project LIST  Projects to migrate (default: sc-bos,gobacnet)
                Format: project or project@branch (e.g., sc-bos@main, myproject@develop)
                Default branches: sc-bos uses @main, gobacnet uses @write
                Custom projects use @latest (current default behavior)
                Can be comma-separated (--project sc-bos,gobacnet) or repeated (--project sc-bos --project gobacnet)
```

## Safety Features

- **Always use `--dry-run` first** to preview changes
- Preserves file permissions and line endings (CRLF vs LF)
- Skips hidden directories (`.git`), build artifacts (`node_modules`, `vendor`, `dist`)
- Processes IDEA run configurations (`.run/`) and GitHub workflows (`.github/`)
- **Skips generated code** - warns you to regenerate them after migration
- **Automatically handles go.mod dependencies** when possible (skips if generated files need regeneration first)
- **Renames files** with `vanti-dev` in their names

## Post-Migration Steps

### After `--preset go`

The tool can automatically update go.mod dependencies using the new organization.

**If you see a warning about generated files**, regenerate them first, then re-run the tool:
```bash
go generate ./...                   # Or run your proto generation script
org-migration --preset go --path .  # Re-run to update go.mod
go build ./...                      # Build and test
```

Otherwise, just build and test your code:
```bash
go build ./...
go test ./...
```


### After `--preset js`

1. Run `npm install` or `yarn install` to fetch updated packages
2. Clear build caches if needed
3. Test your frontend builds

### After `--preset docker`

1. Update CI/CD pipelines that reference image names
2. Test Docker builds
3. Update deployment configs (Kubernetes, docker-compose, etc.)

### After `--preset all`

1. Test your complete application
2. Search for any remaining hardcoded references: `git grep vanti-dev`
3. Commit changes: `git commit -am "chore: migrate to smart-core-os organization"`

## IntelliJ IDEA & GitHub Workflows Support

The tool automatically updates:
- **IDEA run configurations** in `.run/*.xml` - updates package paths
- **GitHub workflow files** in `.github/workflows/` - updates Go build commands
- **File renaming** - renames files with `vanti-dev` in their names

No special flags needed - these are updated automatically when using the `go` or `all` presets.

## Migration Strategy

For a staged migration (recommended):

1. **First**: If your project imports Go packages from `sc-bos`
   ```bash
   org-migration --dry-run --preset go --path .
   org-migration --preset go --path .
   
   # If the tool warned about generated files:
   go generate ./...  # Or run your proto generation
   org-migration --preset go --path .  # Re-run to update go.mod
   
   # Build and test
   go build ./...
   ```

2. **Then**: If your project uses npm packages from `@vanti-dev/sc-bos`
   ```bash
   org-migration --dry-run --preset js --path .
   org-migration --preset js --path .
   npm install && npm run build
   ```

3. **Then**: If your project uses Docker images from `ghcr.io/vanti-dev/sc-bos`
   ```bash
   org-migration --dry-run --preset docker --path .
   org-migration --preset docker --path .
   docker-compose build
   ```

4. **Finally**: Update documentation links
   ```bash
   org-migration --preset docs --path .
   ```

Or for a complete migration in one step:

```bash
org-migration --dry-run --preset all --path .
org-migration --preset all --path .
```

