# Mizito Forwarder

A GoLang service that receives Gotify notifications and forwards them to Mizito chat API.

## Features

- 🚀 Receives Gotify notifications via HTTP API
- 🔐 Automatic authentication with Mizito API
- 💾 JWT token management with persistent storage
- 📝 Comprehensive logging
- 🐳 Docker support
- 🔄 Automatic token refresh on unauthorized errors

## Quick Start

### Prerequisites

- Go 1.24.4 or higher
- Docker and Docker Compose (optional)

### Configuration

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your Mizito credentials:
   ```env
   MIZITO_USERNAME=your_email@example.com
   MIZITO_PASSWORD=your_password
   MIZITO_DIALOG_ID=your_dialog_id
   MIZITO_FROM_USER_ID=your_user_id
   ```

### Running Locally

1. Install dependencies:
   ```bash
   go mod tidy
   ```

2. Build and run:
   ```bash
   go build -o mizito-forwarder
   ./mizito-forwarder
   ```

### Running with Docker

1. Create your `.env` file with the required configuration.

2. Build and run with Docker Compose:
   ```bash
   docker-compose up -d
   ```

3. Check logs:
   ```bash
   docker-compose logs -f mizito-forwarder
   ```

## API Endpoints

### Send Gotify Notification
```http
POST /api/v1/message
Content-Type: application/json

{
  "title": "Deployment Notification",
  "message": "Application deployed successfully",
  "priority": 5,
  "extras": {
    "client::display": {
      "contentType": "text/plain"
    }
  }
}
```

### Health Check
```http
GET /api/v1/health
```

## Configuration Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVER_PORT` | HTTP server port | `:3000` | No |
| `MIZITO_USERNAME` | Mizito username/email | - | Yes |
| `MIZITO_PASSWORD` | Mizito password | - | Yes |
| `MIZITO_DIALOG_ID` | Target dialog ID | - | Yes |
| `MIZITO_FROM_USER_ID` | Your user ID | - | Yes |
| `MIZITO_LOGIN_CODE` | Login code (optional) | `null` | No |
| `MIZITO_REG_ID` | Registration ID (optional) | `null` | No |
| `JWT_TOKEN_FILE` | Token storage file | `token.json` | No |
| `LOG_LEVEL` | Logging level | `info` | No |

## Project Structure

```
MizitoForwarder/
├── config/           # Configuration management
├── handler/          # HTTP request handlers
├── jwt/             # JWT token management
├── logger/          # Logging utilities
├── mizito/          # Mizito API client
├── main.go          # Application entry point
├── Dockerfile       # Docker image definition
├── docker-compose.yml # Docker Compose configuration
├── .env.example     # Environment variables template
└── .gitignore       # Git ignore rules
```

## How It Works

1. **Startup**: The service loads configuration from environment variables
2. **Authentication**: On startup, it attempts to authenticate with Mizito API
3. **Token Storage**: JWT tokens are stored in `token.json` for persistence
4. **API Handling**: Receives Gotify notifications via HTTP POST
5. **Message Forwarding**: Forwards notifications to Mizito chat API
6. **Token Refresh**: Automatically refreshes JWT tokens when they expire

## Development

### Building

```bash
go build -o mizito-forwarder
```

### Testing

```bash
go test ./...
```

### Linting

```bash
go vet ./...
go fmt ./...
```

## Environment Variables

All configuration is managed through environment variables. See `.env.example` for a complete list of available options.

## Logging

The service provides comprehensive logging at different levels:
- `DEBUG`: Detailed debugging information
- `INFO`: General operational information
- `WARN`: Warning messages
- `ERROR`: Error messages

## Security Notes

- JWT tokens are stored in a JSON file with restricted permissions (0600)
- Environment variables are used for sensitive configuration
- API requests include proper headers and authentication
- Token refresh is handled automatically

## Troubleshooting

1. **Authentication Failed**: Check your Mizito credentials in `.env`
2. **Token Expired**: The service will automatically refresh tokens
3. **API Errors**: Check logs for detailed error messages
4. **Port Already in Use**: Change `SERVER_PORT` in your `.env` file

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License.