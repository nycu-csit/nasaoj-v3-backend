.PHONY: up build

up:
	COMPOSE_BAKE=true docker compose up

build:
	COMPOSE_BAKE=true docker compose build --no-cache
