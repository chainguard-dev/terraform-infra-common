.PHONY: help run-local test-local init plan apply destroy clean logs

# Default target
help:
	@echo "AWS Regional Go Service - Example Makefile"
	@echo ""
	@echo "Local Development:"
	@echo "  make run-local       Run the app locally on port 8080"
	@echo "  make test-local      Test the local app endpoints"
	@echo ""
	@echo "Terraform Operations:"
	@echo "  make init            Initialize Terraform"
	@echo "  make plan            Show Terraform execution plan"
	@echo "  make apply           Deploy to AWS"
	@echo "  make destroy         Destroy all resources"
	@echo ""
	@echo "Deployment Testing:"
	@echo "  make test-deployed   Test deployed service endpoints"
	@echo "  make logs            View CloudWatch logs"
	@echo "  make status          Show service status"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean           Clean temporary files"

# Local development
run-local:
	@echo "Starting local server..."
	@cd app && go run main.go

test-local:
	@echo "Testing local endpoints..."
	@echo "\n==> GET /"
	@curl -s http://localhost:8080/ | jq
	@echo "\n==> GET /health"
	@curl -s http://localhost:8080/health | jq
	@echo "\n==> GET /info"
	@curl -s http://localhost:8080/info | jq

# Terraform operations
init:
	@echo "Initializing Terraform..."
	terraform init

plan:
	@echo "Running Terraform plan..."
	terraform plan

apply:
	@echo "Deploying to AWS..."
	terraform apply

destroy:
	@echo "Destroying AWS resources..."
	terraform destroy

# Test deployed service
test-deployed:
	@echo "Testing deployed service..."
	@SERVICE_URL=$$(terraform output -json service_urls 2>/dev/null | jq -r '."us-east-1"'); \
	if [ -z "$$SERVICE_URL" ] || [ "$$SERVICE_URL" = "null" ]; then \
		echo "Error: Could not get service URL. Run 'make apply' first."; \
		exit 1; \
	fi; \
	echo "\n==> Testing $$SERVICE_URL"; \
	echo "\n==> GET /"; \
	curl -s $$SERVICE_URL/ | jq; \
	echo "\n==> GET /health"; \
	curl -s $$SERVICE_URL/health | jq; \
	echo "\n==> GET /info"; \
	curl -s $$SERVICE_URL/info | jq; \
	echo "\n\n==> Open web UI: $$SERVICE_URL/ui"

# View logs
logs:
	@echo "Tailing CloudWatch logs for us-east-1..."
	aws logs tail /aws/apprunner/my-go-service --follow --region us-east-1

# Show service status
status:
	@echo "Service Status:"
	@terraform output -json service_statuses 2>/dev/null | jq || echo "Run 'terraform apply' first"
	@echo "\nService URLs:"
	@terraform output -json service_urls 2>/dev/null | jq || echo "Run 'terraform apply' first"

# Clean up
clean:
	@echo "Cleaning temporary files..."
	rm -rf .terraform .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
	@echo "Done!"
