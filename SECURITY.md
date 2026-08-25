# Security Policy

## Credentials

MSD handles credentials in one place:

- **Site account tokens** — some sites (e.g. Gofile) accept an optional account token to unlock content that guest access cannot reach

### Best practices

- Use the `MSD_GOFILE_TOKEN` (or `GOFILE_TOKEN`) environment variable instead of putting your token in `config.yaml`. Config files on disk are readable by any process running as your user.
- Never commit `config.yaml` to version control. The `.gitignore` does not cover it since it lives outside the repo (XDG config directory), but be careful if you copy it into the project.

## Network

MSD downloads from live, untrusted third-party file-hosting sites: it resolves album/folder/creator/post pages into file lists, then fetches the resulting (often signed, short-lived) CDN links directly to disk. Every URL and filename involved originates from a site MSD does not control.

MSD makes outbound HTTP requests only to:

- The site you point it at (album/folder/creator/post pages and their CDN hosts)
- Optionally, the site's own API if a site requires it to enumerate content (e.g. Redgifs)

No data is sent to any analytics services, or any other third party.

## Reporting a vulnerability

If you find a security issue, please open a GitHub issue or email the maintainer directly. There is no bug bounty program.
