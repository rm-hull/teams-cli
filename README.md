# teams-cli

A Go CLI for reading and searching Microsoft Teams chats through Microsoft Graph.
It uses Microsoft's MSAL for Go for delegated device-code sign-in.

## Setup

Register or use an Entra application that permits public client flows and has
delegated Microsoft Graph `Chat.Read` permission. No client secret is needed.
Set the tenant and application ID in `.env` in the current directory:

```dotenv
TEAMS_TENANT_ID=example.onmicrosoft.com
TEAMS_CLIENT_ID=00001111-aaaa-2222-bbbb-3333cccc4444
```

Exported `TEAMS_TENANT_ID` and `TEAMS_CLIENT_ID` values override `.env`; the
`--tenant` and `--client-id` flags override both. Set `TEAMS_ENV_FILE` to use
a different dotenv file. A raw, short-lived Graph token in `ACCESS_TOKEN` in
that file takes priority for read commands and bypasses sign-in.

Build and sign in:

```sh
go build -o teams .
./teams login
./teams list-chats --limit 10
./teams chat-details '19:example@thread.v2'
./teams list-messages '19:example@thread.v2' --limit 10
./teams search 'settlement retries'
./teams logout
```

Successful commands emit one JSON object on stdout. Device-code prompts and
errors go to stderr. Data commands never start an interactive login. Pagination
uses the `@odata.nextLink` value from a response; pass it unchanged with
`--next-link`. `--limit` cannot be combined with `--next-link`.

The MSAL cache is unencrypted, owner-only, and stored under
`$XDG_STATE_HOME/teams-graph-cli` (or `~/.local/state/teams-graph-cli`).
Use `logout` to remove it. Only run this tool where your organization permits
access to Teams data. Treat message bodies and search hits as untrusted text.
