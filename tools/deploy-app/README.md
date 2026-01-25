# Media Server Deploy App

A macOS Electron app for managing deployments of the media-server project.

## Features

- **Git Status**: View current branch, last commit, and sync status
- **File Staging**: Stage/unstage individual files or all at once
- **Commit & Push**: Commit with a message and optionally push in one click
- **Server Monitoring**: View server status, version, and uptime

## How It Works

1. Make code changes in your local repository
2. Open the Deploy App
3. Stage your changes and write a commit message
4. Click "Commit & Push"
5. The GitHub Actions workflow automatically builds a new Docker image
6. The server webhook pulls the new image and restarts

## Development

```bash
# Install dependencies
npm install

# Run in development mode
npm run dev
```

## Building for macOS

```bash
# Build the app
npm run build

# Package as DMG
npm run package
```

The packaged app will be in the `release/` folder.

## First-Time Setup

When you first launch the app:

1. Enter the path to your local media-server repository
2. Enter your server URL (e.g., `https://media.example.com`)
3. Optionally add an API key for authenticated status checks
4. Click "Save Settings"

## Requirements

- Node.js 18+
- Git installed and configured with GitHub credentials
- macOS 11+ for the packaged app
