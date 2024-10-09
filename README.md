# Smart Dictionary

The **Smart Dictionary** is a language-learning tool designed to provide real-time translations, word suggestions, and definitions, helping users improve their vocabulary while engaging with foreign texts. Built in Go, this project offers an easy-to-use API for word translations and personalized vocabulary tracking, powered by external translation APIs.

## Features

- **Real-time Translation**: Context-aware translation for multiple languages.
- **Word Definitions**: Fetch definitions for selected words in a variety of languages.
- **Token-Based Authentication**: JWT-based authentication for secure API access.
- **Integration with GPT and External APIs**: Leverages GPT models and translation services for high-quality translations.
- **Dockerized Deployment**: Easily build and run the application with Docker.

## Folder Structure

```bash
smart-dictionary/
├── Dockerfile                  # Docker image configuration
├── README.md                   # Project documentation
├── cmd/
│   └── main.go                 # Main application entry point
├── db/
│   └── init.sql                # Database initialization script
├── docker-compose.yaml         # Docker Compose configuration
├── go.mod                      # Go module file
├── go.sum                      # Go module dependencies
└── internal/
    ├── domain/                 # Core domain logic
    │   ├── auth.go             # Authentication domain logic
    │   ├── chat_gpt.go         # GPT model integration
    │   ├── languages.go        # Supported languages definitions
    │   ├── token.go            # JWT token handling
    │   ├── translation.go      # Translation domain logic
    │   └── translation_request.go # Translation request handling
    ├── infrastructure/         # Repositories and external service integration
    │   ├── auth_repository.go  # User authentication repository
    │   ├── mochi_api.go        # Mochi API integration for translations
    │   └── translator_repository.go # Translation repository
    ├── middleware/             # Middleware for handling requests
    │   └── auth.go             # JWT authentication middleware
    ├── server/                 # Server-related code
    │   └── translator_server.go # HTTP server and API routes
    └── usecase/                # Application services and business logic
        ├── auth_service.go     # Authentication service
        └── jwt.go              # JWT-related business logic
```

## Getting Started

### Prerequisites

To run this project locally, you'll need:

- **Go** (version 1.18 or higher)
- **Docker** (for containerized deployment)
- **PostgreSQL** (as the database engine)

### Installation

1. **Clone the repository**:

   ```bash
   git clone https://github.com/bukhavtsov/smart-dictionary.git
   ```

2. **Navigate into the project directory**:

   ```bash
   cd smart-dictionary
   ```

3. **Set up the environment**:

   Configure your environment variables by creating a `.env` file or using the provided `.env.tmp` as a template.

### Running the Application

You can run the application either directly or using Docker.

#### 1. Run Locally with Go

Make sure to have the necessary Go modules installed:

```bash
go mod tidy
```

Then, run the application:

```bash
go run cmd/main.go
```

The app will be available at `http://localhost:8080`.

#### 2. Run via Docker

Build the Docker image:

```bash
docker build -t smart-dict .
```

Run the application using Docker:

```bash
docker run --env-file .env.tmp -p 8080:8080 smart-dict
```

Alternatively, use `docker-compose` to spin up the application and any dependencies (e.g., database):

```bash
docker-compose --env-fili .env.tmp up -d
```

### Database Initialization

The application uses PostgreSQL as its database. You can initialize the database schema with the SQL file provided in the `db/` folder. If you're using Docker, this is handled automatically by the `docker-compose.yaml` file.

For manual setup:

1. Create a PostgreSQL database.
2. Run the `db/init.sql` script to create necessary tables.

## API Endpoints

The application exposes various endpoints for translations and user management. Below are key endpoints:

- **POST** `/translate` - Translate a word or phrase.
- **POST** `/login` - User authentication.
- **POST** `/register` - User registration.
- **GET** `/languages` - Fetch supported languages for translation.

## Testing

To run the tests:

```bash
go test ./...
```

Make sure that your testing environment is properly set up with mock services or a test database.

## Docker

### Build and Run the Docker Image

To build the Docker image:

```bash
docker build -t smart-dict .
```

Run the application using the built image:

```bash
docker run --env-file .env.tmp -p 8080:8080 smart-dict
```

### Using Docker Compose

You can also run the application with `docker-compose`, which will handle the database and any additional services:

```bash
docker-compose up --build
```

### Environment Variables

Make sure to configure the `.env` file with the necessary environment variables, such as:

- `DB_HOST`: Database host
- `DB_USER`: Database user
- `DB_PASSWORD`: Database password
- `JWT_SECRET`: Secret for JWT token signing

## Contributing

We welcome contributions! To contribute to the Smart Dictionary project:

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/new-feature`).
3. Make your changes and commit them (`git commit -m 'Add new feature'`).
4. Push your branch (`git push origin feature/new-feature`).
5. Open a pull request.

Please ensure that your code adheres to the project guidelines and includes relevant tests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contact

For any issues or suggestions, feel free to create an issue in the repository or reach out to the maintainers.
