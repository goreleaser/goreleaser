# Scripts

This directory contains various utility scripts for the GoReleaser project.

## update-sponsors.py

Updates both OpenCollective and GitHub Sponsors lists in `www/docs/sponsors.md` and `README.md`.

**Usage:**
```bash
# Update OpenCollective sponsors only (no token needed)
python3 scripts/update-sponsors.py

# Update both OpenCollective and GitHub Sponsors
GITHUB_TOKEN=your_token python3 scripts/update-sponsors.py

# Or via task
task docs:sponsors

# With GitHub token
GITHUB_TOKEN=your_token task docs:sponsors
```

**What it does:**
- Fetches active sponsors and backers from OpenCollective API
- Fetches active GitHub Sponsors for [@caarlos0](https://github.com/caarlos0) (if GITHUB_TOKEN is set)
- Filters for recurring contributions (monthly or yearly) only
- Groups sponsors into tiers based on their monthly contribution amount:
  - **Gold Sponsors**: $100+ per month (with logo)
  - **Silver Sponsors**: $50-99 per month (with logo)
  - **Bronze Sponsors**: $20-49 per month (with logo)
  - **Backers**: <$20 per month (text list only for docs, images for README)
- Deduplicates sponsors who appear multiple times
- Updates both `www/docs/sponsors.md` and `README.md` with the latest data
- Uses HTML comment markers for easy updates:
  - `<!-- opencollective:begin/end -->` for OpenCollective section
  - `<!-- github-sponsors:begin/end -->` for GitHub Sponsors section

**Requirements:**
- Python 3.6+
- No external dependencies (uses standard library only)
- `GITHUB_TOKEN` environment variable (optional, for GitHub Sponsors)
  - In CI: automatically provided by GitHub Actions
  - Locally: use a personal access token with `read:user` scope

**Note:** 
- OpenCollective data is public and requires no authentication
- GitHub Sponsors data requires a GitHub token with appropriate permissions

## Other Scripts

- `get-releases.sh` - Fetches release information from GitHub API
- `completions.sh` - Generates shell completions
- `manpages.sh` - Generates man pages
- `cmd_docs.sh` - Generates command documentation
- And more...

Run `task -l` to see all available tasks.
