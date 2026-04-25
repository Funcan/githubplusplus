# gh++

The `gh` CLI from github has loads of great features, but it lacks some
desirable features.

`gh++` command-line tool for managing GitHub repos at scale - across your
personal account and multiple orgs.


## Requirements

- [gh CLI](https://cli.github.com/) - authentication is read from `gh auth 
  login`; no separate setup needed.
- Go 1.21+ (to build from source)

## Installation

```sh
git clone https://github.com/Funcan/githubplusplus
cd githubplusplus
make install
```

Or just build locally:

```sh
make build
./gh++ --help
```

## Global Flags

These flags apply to all commands:

| Flag | Description |
|---|---|
| `--user USER` | Target a personal account (defaults to the authenticated user 
if no `--org` is given) |
| `--org ORG` | Target a GitHub org or user; can be specified multiple times |

If a name given to `--org` is not found as an org, it is automatically retried 
as a user account.

## Commands

### `list`

List repos matching the given filters.

```sh
# List your own repos
gh++ list

# List all repos for a user
gh++ list --user Funcan

# List all repos across multiple orgs
gh++ list --org my-org --org another-org

# List only forks
gh++ list --forks

# List forks for a specific user
gh++ list --user Funcan --forks

# List non-archived Go repos
gh++ list --language Go --no-archived

# Output as JSON (one object per line)
gh++ list --format json
```

**Flags:**

| Flag | Description |
|---|---|
| `--forks` | Include only forked repos |
| `--no-forks` | Exclude forked repos |
| `--language LANG` | Filter by primary language (case-insensitive) |
| `--archived` | Include only archived repos |
| `--no-archived` | Exclude archived repos |
| `--format table\|json` | Output format (default: `table`) |

---

### `clone`

Clone all matching repos into a local directory. Uses the same filters as 
`list`.

```sh
# Clone all your repos into the current directory
gh++ clone

# Clone all repos from an org into ~/code/my-org
gh++ clone --org my-org --dest ~/code/my-org

# Clone only forks, skip any already cloned
gh++ clone --forks --skip-existing

# Clone Go repos from multiple orgs with 16 parallel workers
gh++ clone --org org-a --org org-b --language Go --concurrency 16

# Clone a user's repos (falls back from org lookup automatically)
gh++ clone --org Funcan --dest ~/code/funcan
```

**Flags:**

| Flag | Description |
|---|---|
| `--forks` | Include only forked repos |
| `--no-forks` | Exclude forked repos |
| `--language LANG` | Filter by primary language (case-insensitive) |
| `--archived` | Include only archived repos |
| `--no-archived` | Exclude archived repos |
| `--dest DIR` | Directory to clone repos into (default: `.`) |
| `--skip-existing` | Skip repos whose local directory already exists |
| `--concurrency N` | Number of parallel clone workers (default: `8`) |

## Development

```sh
make build    # build ./gh++ binary
make test     # run tests with coverage
make fmt      # format source with gofmt
make install  # install to $GOPATH/bin
make clean    # remove built binary
```
