start:
	docker compose -f "docker-compose.yml" up -d --build 

first_start:
	docker compose -f "docker-compose.yml" up -d --build  && migrate -path ./schema -database 'postgres://postgres:qwerty@0.0.0.0:5432/postgres?sslmode=disable' up

stop:
	docker compose -f "docker-compose.yml" down 

restart:
	docker compose -f "docker-compose.yml" down && docker compose -f "docker-compose.yml" up -d --build

migrate:
	migrate -path ./schema -database 'postgres://postgres:qwerty@0.0.0.0:5432/postgres?sslmode=disable' up

test:
	go test ./... -cover

free:
	sudo lsof -i :5432
