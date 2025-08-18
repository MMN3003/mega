# 1. Start local DB + app
make compose-up

# 2. Build the binary
make build

# 3. Run locally
make run

# 4. Lint & test
make lint
make test

# 5. Swagger docs
make swagger

# 6. Build Docker image
make docker-build
