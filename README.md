# Discord Message Cleaner

A Go utility to bulk delete Discord messages from a channel using the Discord Bot API.

## Features

- Fetch messages from a Discord channel
- Preview messages before deletion
- Interactive confirmation before deleting
- Rate limit handling with automatic retry
- Option to start with oldest messages

## Prerequisites

- Go 1.22 or higher
- Discord Bot Token with message deletion permissions
- Channel ID and Server ID

## Installation

```bash
go build
```

## Usage

1. Create a `.env` file based on `example.env`:

```
DISCORD_TOKEN=your_bot_token_here
SERVER_ID=your_server_id
CHANNEL_ID=your_channel_id
START_WITH_OLDEST=true  # or false to start with newest
```

2. Run the application:

```bash
./Discord-Message-Cleaner
```

3. Review the messages that will be deleted
4. Confirm the deletion when prompted

## Configuration

Environment variables:

- `DISCORD_TOKEN`: Your Discord Bot token (required)
- `SERVER_ID`: The ID of the server/guild (required)
- `CHANNEL_ID`: The ID of the channel to clean (required)
- `START_WITH_OLDEST`: Set to `true` to process oldest messages first, `false` for newest (optional, default: false)

## Notes

- The application fetches up to 100 messages at a time
- Rate limits are automatically handled
- Messages are deleted with the reason "Pepe-Deletor" by default
