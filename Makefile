#изменить имя бинарника
build:
	go build -o backend/bin/amazing-helper backend/cmd/main.go
run: build

	./backend/bin/amazing-helper --config=./backend/configs/local.yaml
run-containers:
	docker compose down -v
	docker compose up -d


build-docker:
	docker build -t ghcr.io/pulse-fetch/gateway:latest .
push-docker: build-docker
	docker push ghcr.io/pulse-fetch/gateway:latest