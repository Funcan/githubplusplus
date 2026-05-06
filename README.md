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

---

### `issues`

List open issues for one or more repos.

```sh
# List issues for the repo in the current directory
gh++ issues

# List issues for a specific repo
gh++ issues owner/repo

# List issues for all repos owned by a user or org
gh++ issues my-org

# List issues for multiple repos
gh++ issues owner/repo-a owner/repo-b
```

Each argument may be a local path to a git checkout, an `owner/repo` reference,
or an `owner` name (which expands to all repos for that user or org).

---

### `prs`

List open pull requests for one or more repos.

```sh
# List PRs for the repo in the current directory
gh++ prs

# List PRs for a specific repo
gh++ prs owner/repo

# List PRs for all repos owned by a user or org
gh++ prs my-org

# List PRs for multiple repos
gh++ prs owner/repo-a owner/repo-b

# List only PRs that are ready to merge (checks passed, no conflicts, reviews approved)
gh++ prs --ready owner/repo
```

Each argument may be a local path to a git checkout, an `owner/repo` reference,
or an `owner` name (which expands to all repos for that user or org).

**Flags:**

| Flag | Description |
|---|---|
| `--ready` | Only list PRs that are ready to merge (checks passed, no merge conflicts, required reviews approved) |

---

### `move`

Move one or more repos to a different org or user account.

```sh
# Move a repo to an org
gh++ move owner/repo --to-org my-org

# Move a repo to a personal account
gh++ move owner/repo --to-user myusername

# Move multiple repos in parallel
gh++ move owner/repo-a owner/repo-b --to-org my-org --concurrency 4
```

**Flags:**

| Flag | Description |
|---|---|
| `--to-org ORG` | Destination org (mutually exclusive with `--to-user`) |
| `--to-user USER` | Destination user (mutually exclusive with `--to-org`) |
| `--concurrency N` | Number of parallel transfer workers (default: `8`) |

---

### `fork`

Subcommands for working with forked repositories.

#### `fork status`

Show whether one or more forks are up to date with their upstream.

```sh
# Check the fork in the current directory
gh++ fork status

# Check a specific fork by owner/repo
gh++ fork status owner/repo

# Check multiple forks, suppress identical ones
gh++ fork status owner/repo-a owner/repo-b --ignore-status identical
```

Possible statuses: `identical`, `behind`, `ahead`, `diverged`. A `PRsOpen`
label is also shown when the fork has open pull requests to its upstream.

**Flags:**

| Flag | Description |
|---|---|
| `--ignore-status STATUS,...` | Comma-separated statuses to suppress (`identical`, `behind`, `ahead`, `diverged`, `PRsOpen`) |

#### `fork update`

Pull upstream changes into one or more forked repositories via the GitHub API.
If a local path is given, also runs `git fetch origin` and fast-forwards the
default branch (when it is currently checked out).

```sh
# Update the fork in the current directory
gh++ fork update

# Update a specific fork
gh++ fork update owner/repo

# Update all forks for an org
gh++ fork update my-org
```

#### `fork prs`

List the URLs of all open pull requests from a fork to its upstream.

```sh
# List PRs for the fork in the current directory
gh++ fork prs

# List PRs for a specific fork
gh++ fork prs owner/repo
```

---

## Development

```sh
make build    # build ./gh++ binary
make test     # run tests with coverage
make fmt      # format source with gofmt
make install  # install to $GOPATH/bin
make clean    # remove built binary
```
