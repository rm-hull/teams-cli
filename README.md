# teams-cli

A command-line tool for reading and searching Microsoft Teams chats through the Microsoft Graph API, predominantly designed for agentic access. It supports delegated device-code sign-in, so no client secret is required.

## Installation

### Homebrew (macOS)

```sh
brew install rm-hull/tap/teams-cli
```

### Build from source

```sh
git clone https://gitlab.com/rm-hull/teams-cli.git
cd teams-cli
make build
```

## Quick Start

```sh
# Sign in (opens a browser for device-code authentication)
teams-cli login

# List your recent chats
teams-cli list-chats --limit 10

# Search across all your messages
teams-cli search "settlement retries"

# When you're done
teams-cli logout
```

## Prerequisites

You need an Azure Entra (formerly Azure AD) application with:

- **Public client flows** enabled (required for device-code sign-in).
- **Delegated `Chat.Read`** permission for Microsoft Graph.

No client secret is needed.

## Configuration

Create a `.env` file in the project directory with your tenant and application IDs:

```dotenv
TEAMS_TENANT_ID=example.onmicrosoft.com
TEAMS_CLIENT_ID=00001111-aaaa-2222-bbbb-3333cccc4444
```

### Configuration priority (highest to lowest)

1. CLI flags (`--tenant`, `--client-id`)
2. Exported environment variables (`TEAMS_TENANT_ID`, `TEAMS_CLIENT_ID`)
3. `.env` file (path customizable via `TEAMS_ENV_FILE`)

### Using a pre-existing token

If you already have a short-lived Microsoft Graph access token, set `ACCESS_TOKEN` in your `.env` file. It takes priority over cached sign-in and bypasses the login flow entirely.

## Commands

| Command | Description |
|---|---|
| `login` | Sign in with a device code (opens browser) |
| `logout` | Remove cached credentials |
| `list-chats` | List your Teams chats |
| `chat-details <chat-id>` | Get metadata for a specific chat |
| `list-messages <chat-id>` | List messages in a chat |
| `search <query>` | Search across all visible chat messages |

### Common flags

| Flag | Applies to | Default | Description |
|---|---|---|---|
| `--limit` | `list-chats`, `list-messages`, `search` | 20 | Number of results (1–50) |
| `--next-link` | `list-chats`, `list-messages` | — | Pagination token from a previous response (`@odata.nextLink`) |
| `--offset` | `search` | 0 | Zero-based result offset for pagination |
| `--tenant` | all | — | Entra directory ID or tenant domain |
| `--client-id` | all | — | Entra application ID |

**Note:** `--limit` and `--next-link` cannot be used together.

## Examples

```sh
# List 10 recent chats
teams-cli list-chats --limit 10

# View details for a specific chat
teams-cli chat-details '19:example@thread.v2'

# List the latest 10 messages in a chat
teams-cli list-messages '19:example@thread.v2' --limit 10

# Search for a keyword
teams-cli search "settlement retries"

# Paginate search results (next 20 after the first 20)
teams-cli search "settlement retries" --offset 20 --limit 20

# Use a custom .env file
TEAMS_ENV_FILE=~/.config/teams/.env teams-cli list-chats
```

## Output

- All successful commands write a single JSON object to **stdout**, making it easy to pipe into other tools (e.g., `jq`).
- Device-code prompts, progress messages, and errors go to **stderr**.
- Data commands (`list-chats`, `chat-details`, etc.) never trigger an interactive login. If you're not signed in, they fail with an error.

## Security

- The MSAL token cache is stored unencrypted with owner-only permissions under `$XDG_STATE_HOME/teams-graph-cli` (defaults to `~/.local/state/teams-graph-cli`).
- Run `logout` to clear cached credentials.
- Only use this tool where your organization permits access to Teams data.
- Treat message bodies and search results as untrusted text.
