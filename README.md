# git-subclone

Manage nested git repositories with independent push capability.

Unlike git submodules or subtrees, subclones are fully independent repositories that:
- Clone automatically when you sync
- Push directly to their own remote
- Keep source files tracked by the parent (but not `.git`)

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew install yejune/tap/git-subclone
```

### Using Go

```bash
go install github.com/yejune/git-subclone@latest
```

### From Source

```bash
git clone https://github.com/yejune/git-subclone.git
cd git-subclone
go build -o git-subclone
sudo mv git-subclone /usr/local/bin/
```

## Quick Start

```bash
# Clone a repository as subclone (just like git clone!)
git subclone https://github.com/user/repo.git

# With custom path
git subclone https://github.com/user/repo.git packages/repo

# With specific branch
git subclone -b develop https://github.com/user/repo.git

# SSH format
git subclone git@github.com:user/repo.git
```

## Usage

### Add a Subclone

```bash
# Quick way (auto-extracts repo name)
git subclone https://github.com/user/lib.git              # → ./lib/
git subclone https://github.com/user/lib.git packages/lib # → ./packages/lib/

# Or use add command
git subclone add https://github.com/user/lib.git packages/lib
git subclone add git@github.com:user/lib.git packages/lib -b develop
```

### Sync All Subclones

```bash
# Clone or pull all registered subclones
git subclone sync

# Recursively sync subclones within subclones
git subclone sync --recursive
```

### List Subclones

```bash
git subclone list
git subclone ls        # alias
git subclone status    # detailed status
```

### Push Changes

```bash
# Push a specific subclone
git subclone push packages/lib

# Push all modified subclones
git subclone push --all
```

### Remove a Subclone

```bash
# Remove and delete files
git subclone remove packages/lib

# Remove from manifest but keep files
git subclone rm packages/lib --keep-files
```

### Install Git Hooks

```bash
# Auto-sync subclones after checkout
git subclone init
```

## How It Works

### Directory Structure

```
mother-project/
├── .git/                    ← Parent project git
├── .subclones.yaml          ← Subclone manifest (tracked by parent)
├── .gitignore               ← Contains "packages/lib/.git/"
├── src/
│   └── main.go
└── packages/
    └── lib/
        ├── .git/            ← Subclone's independent git
        └── lib.go           ← Tracked by BOTH repos
```

### Key Points

1. **Independent Git**: Each subclone has its own `.git` directory
2. **Source Tracking**: Parent tracks subclone's source files (not `.git`)
3. **Direct Push**: `cd packages/lib && git push` goes to lib's remote
4. **Manifest File**: `.subclones.yaml` records all subclones

### Manifest Format

```yaml
subclones:
  - path: packages/lib
    repo: https://github.com/user/lib.git
    branch: main
  - path: packages/utils
    repo: git@github.com:user/utils.git
```

## Comparison

|                    | Submodule          | Subtree            | Subclone           |
|--------------------|--------------------|--------------------|--------------------|
| Clone simplicity   | ❌ --recursive     | ✅                 | ✅ (with hook)     |
| Push intuitive     | ✅                 | ❌ special command | ✅                 |
| Files in parent    | ❌ pointer only    | ✅                 | ✅                 |
| Clear management   | ✅ .gitmodules     | ❌                 | ✅ .subclones.yaml |
| Independent repo   | ✅                 | ❌                 | ✅                 |

## License

MIT
