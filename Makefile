VERSION ?= dev
LDFLAGS := -s -w

IMAGE_REGISTRY ?= reg.meh.wf
IMAGE_NAME     ?= voltpilot
IMAGE_TAG      ?= dev
DEPLOY_NS      ?= voltpilot
INGRESS_HOST   ?=
KUBE_CONTEXT   ?=
KUBECTL_CTX    := $(if $(KUBE_CONTEXT),--context $(KUBE_CONTEXT),)

.PHONY: help build build-prod fmt vet test coverage lint helm-lint goreleaser-check clean dev-deploy-k8s

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the voltpilot binary (without embedded frontend)
	@go build -ldflags "$(LDFLAGS)" -o bin/voltpilot ./cmd/server

build-prod: ## Build the production binary with the SvelteKit frontend embedded
	@cd web && npm ci --silent && npm run build
	@go build -ldflags "$(LDFLAGS)" -tags prodfrontend -o bin/voltpilot ./cmd/server

fmt: ## Run go fmt
	go fmt ./...

vet: ## Run go vet
	go vet ./...

test: ## Run all Go tests with race detector and coverage
	@go test -race -coverprofile=coverage.out ./...

coverage: test ## Print coverage by func and total
	@go tool cover -func=coverage.out

helm-lint: ## Lint the Helm chart
	@if [ -d charts/voltpilot ]; then helm lint ./charts/voltpilot; else echo "charts/voltpilot not present - skipping"; fi

goreleaser-check: ## Validate .goreleaser.yaml
	@if [ -f .goreleaser.yaml ]; then goreleaser check; else echo ".goreleaser.yaml not present - skipping"; fi

lint: fmt vet helm-lint goreleaser-check ## Run linters

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out dist/ web/build/

dev-deploy-k8s: ## Build dev image, push to IMAGE_REGISTRY, deploy to K8s namespace DEPLOY_NS
	@if [ -z "$(IMAGE_REGISTRY)" ]; then echo "ERROR: IMAGE_REGISTRY is not set. See AGENTS.md.local."; exit 1; fi
	@if [ -z "$(INGRESS_HOST)" ]; then echo "ERROR: INGRESS_HOST is not set. See AGENTS.md.local."; exit 1; fi
	@echo "Building dev image..."
	@docker build --target app -t $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) -f Dockerfile.dev .
	@echo "Pushing to $(IMAGE_REGISTRY)..."
	@docker push $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	@IMAGE_DIGEST=$$(docker inspect $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) --format='{{index .RepoDigests 0}}' | cut -d'@' -f2); \
	echo "Using digest: $$IMAGE_DIGEST"; \
	echo "Ensuring namespace $(DEPLOY_NS) exists..."; \
	kubectl $(KUBECTL_CTX) get namespace $(DEPLOY_NS) >/dev/null 2>&1 || kubectl $(KUBECTL_CTX) create namespace $(DEPLOY_NS); \
	echo "Deploying to namespace $(DEPLOY_NS)..."; \
	kubectl $(KUBECTL_CTX) -n $(DEPLOY_NS) delete deploy/voltpilot --ignore-not-found; \
	helm template voltpilot ./charts/voltpilot \
		--namespace $(DEPLOY_NS) \
		--set image.repository="$(IMAGE_REGISTRY)/$(IMAGE_NAME)" \
		--set image.tag="$(IMAGE_TAG)" \
		--set image.digest="$$IMAGE_DIGEST" \
		--set ingress.hosts[0]="$(INGRESS_HOST)" \
		--set ingress.tls[0].hosts[0]="$(INGRESS_HOST)" \
		$(HELM_EXTRA_ARGS) \
	| kubectl $(KUBECTL_CTX) apply -n $(DEPLOY_NS) -f - --wait
	@echo "Deployment dispatched. Watch:"
	@echo "  kubectl $(KUBECTL_CTX) -n $(DEPLOY_NS) rollout status deploy/voltpilot"
