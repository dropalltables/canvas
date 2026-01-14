# canvas

CLI for Canvas LMS.

## Install

```
go build -o canvas .
```

## Usage

```
canvas              # Interactive TUI
canvas a            # List assignments
canvas a -f json    # JSON output
canvas a -c math    # Filter by course
canvas auth login   # Authenticate
```

## Flags

```
-f, --format   table, json, compact
-p, --past     Days in past (default 7)
-F, --future   Days in future (default 30)
-a, --all      Include completed
-c, --course   Filter by course name
-t, --type     assignment, quiz, discussion
```

## Config

Credentials stored at `~/.config/canvas/config.json` or via `CANVAS_BASE_URL` and `CANVAS_TOKEN` env vars.
