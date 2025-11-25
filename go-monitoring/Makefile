.PHONY: help build run clean test deps build-prod

help: ## Affiche l'aide
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

deps: ## Télécharge les dépendances
	@echo "📦 Téléchargement des dépendances..."
	go mod download
	go mod tidy

build: deps ## Compile l'application
	@echo "🔨 Compilation de l'application..."
	go build -o bin/monitoring cmd/server/main.go
	@echo "✅ Binaire créé : bin/monitoring"

build-prod: deps ## Compile pour production (optimisé)
	@echo "🔨 Compilation pour production..."
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/monitoring cmd/server/main.go
	@echo "✅ Binaire optimisé créé : bin/monitoring"

run: build ## Lance le serveur
	@echo "🚀 Démarrage du serveur..."
	./bin/monitoring

dev: ## Lance en mode développement (avec auto-reload)
	@echo "🔧 Mode développement..."
	go run cmd/server/main.go

clean: ## Nettoie les fichiers compilés
	@echo "🧹 Nettoyage..."
	rm -rf bin/
	go clean

test: ## Lance les tests
	@echo "🧪 Lancement des tests..."
	go test -v ./...

fmt: ## Formate le code
	@echo "✨ Formatage du code..."
	go fmt ./...

lint: ## Vérifie le code
	@echo "🔍 Vérification du code..."
	go vet ./...

docker-build: ## Construit l'image Docker
	@echo "🐳 Construction de l'image Docker..."
	docker build -t monitoring-bornes:latest .

docker-run: docker-build ## Lance le container Docker
	@echo "🐳 Lancement du container..."
	docker run -p 8080:8080 monitoring-bornes:latest
