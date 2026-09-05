systemctl start docker

docker compose up -d

export $(cat .env | xargs)

/usr/local/go/bin/go run cmd/main.go