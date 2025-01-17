.PHONY: build
build:
	go build -o ./build/tm2-indexer ./cmd/tm2-indexer 

psql:
	docker compose exec -it postgres psql tm2-indexer