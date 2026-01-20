# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

.PHONY: help init plan apply destroy test-local clean outputs

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

init: ## Initialize Terraform
	terraform init

plan: ## Run terraform plan
	terraform plan -out=plan.out

apply: plan ## Apply the Terraform configuration
	terraform apply plan.out

destroy: ## Destroy all resources
	terraform destroy

test-local: ## Test the prober application locally
	@echo "Starting prober locally on port 8080..."
	@echo "Testing with target: https://httpbin.org/status/200"
	cd app && \
	AUTHORIZATION="test-secret-123" \
	TARGET_URL="https://httpbin.org/status/200" \
	PORT="8080" \
	go run main.go

test-deployed: ## Test the deployed prober
	@echo "Testing deployed prober..."
	@PROBER_URL=$$(terraform output -raw prober_url 2>/dev/null); \
	AUTH_SECRET=$$(terraform output -raw authorization_secret 2>/dev/null); \
	if [ -z "$$PROBER_URL" ] || [ -z "$$AUTH_SECRET" ]; then \
		echo "Error: Prober not deployed or outputs not available."; \
		echo "Run 'make apply' first."; \
		exit 1; \
	fi; \
	echo "Prober URL: $$PROBER_URL"; \
	echo ""; \
	echo "Testing with correct authorization:"; \
	curl -s -H "Authorization: $$AUTH_SECRET" "$$PROBER_URL/" | jq .; \
	echo ""; \
	echo "Testing with incorrect authorization (should fail):"; \
	curl -s -w "\nHTTP Status: %{http_code}\n" -H "Authorization: wrong-secret" "$$PROBER_URL/"

outputs: ## Show all Terraform outputs
	terraform output

view-canary: ## View canary status in AWS Console
	@CANARY_NAME=$$(terraform output -raw canary_name 2>/dev/null); \
	REGION=$$(terraform output -raw region 2>/dev/null); \
	if [ -z "$$CANARY_NAME" ] || [ -z "$$REGION" ]; then \
		echo "Error: Canary not deployed or outputs not available."; \
		exit 1; \
	fi; \
	echo "Opening CloudWatch Synthetics console..."; \
	open "https://console.aws.amazon.com/cloudwatch/home?region=$$REGION#synthetics:canary/detail/$$CANARY_NAME"

view-logs: ## View App Runner logs
	@SERVICE_NAME=$$(terraform output -raw prober_name 2>/dev/null); \
	REGION=$$(terraform output -raw region 2>/dev/null); \
	if [ -z "$$SERVICE_NAME" ] || [ -z "$$REGION" ]; then \
		echo "Error: Service not deployed or outputs not available."; \
		exit 1; \
	fi; \
	echo "Fetching logs for service: $$SERVICE_NAME"; \
	aws logs tail /aws/apprunner/$$SERVICE_NAME --follow --region $$REGION

view-metrics: ## View canary metrics
	@CANARY_NAME=$$(terraform output -raw canary_name 2>/dev/null); \
	REGION=$$(terraform output -raw region 2>/dev/null); \
	if [ -z "$$CANARY_NAME" ] || [ -z "$$REGION" ]; then \
		echo "Error: Canary not deployed or outputs not available."; \
		exit 1; \
	fi; \
	echo "Fetching canary metrics for last hour..."; \
	aws cloudwatch get-metric-statistics \
		--namespace CloudWatchSynthetics \
		--metric-name SuccessPercent \
		--dimensions Name=CanaryName,Value=$$CANARY_NAME \
		--start-time $$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
		--end-time $$(date -u +%Y-%m-%dT%H:%M:%S) \
		--period 300 \
		--statistics Average \
		--region $$REGION \
		--output table

validate: ## Validate Terraform configuration
	terraform validate
	terraform fmt -check

format: ## Format Terraform files
	terraform fmt -recursive

clean: ## Clean up local files
	rm -f plan.out
	rm -f .terraform.lock.hcl
	rm -rf .terraform

.DEFAULT_GOAL := help
