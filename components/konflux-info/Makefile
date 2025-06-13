BANNER_SCHEMA := $(abspath validator/banner-schema.json)
PROD_DIR := $(abspath production)
STAGING_DIR := $(abspath staging)

.PHONY: verify-all verify-production verify-staging verify-banner verify-kustomize

define verify_banner
	@echo ">>> Verifying banner content in $(1)..."
	@./validator/validator $(BANNER_SCHEMA) $(1)
endef

define verify_kustomize
	@for d in $(1)/*/ ; do \
	  echo ">>> Verifying kustomize build $$d"; \
	  if kustomize build "$$d" >/dev/null; then \
	    echo "✔ $$d OK"; \
	  else \
	    echo "❌ $$d FAILED"; \
	    exit 1; \
	  fi; \
	done
endef

build-validator:
	@echo "Building validator..."
	go build -o validator validator/validator.go

verify-production-banner: 
	$(call verify_banner,$(PROD_DIR))

verify-production-kustomize:
	$(call verify_kustomize,production)

verify-staging-banner:
	$(call verify_banner,$(STAGING_DIR))

verify-staging-kustomize:
	$(call verify_kustomize,staging)

verify-production: verify-production-banner verify-production-kustomize
verify-staging: verify-staging-banner verify-staging-kustomize

verify-all: verify-production verify-staging
