# PRPilot: AI Pull Request Review Assistant

PRPilot is a small serverless developer-tooling project built with Go, AWS Lambda, API Gateway, Amazon Bedrock, DynamoDB, and the GitHub API. It accepts a pull request diff, runs it through a multi-stage AI review pipeline, returns structured JSON feedback, and stores the review for later retrieval.

## Architecture

```text
Client or GitHub Webhook
        |
        v
API Gateway HTTP API
        |
        v
Go Lambda Function
        |
        |----> Amazon Bedrock Runtime
        |----> DynamoDB: prpilot_reviews
        |----> CloudWatch Logs
        |----> GitHub API, optional webhook flow
```

## Features

- Go backend with local HTTP mode and Lambda mode
- Multi-stage AI review pipeline:
  - Summarizer agent
  - Risk analysis agent
  - Security review agent
  - Test suggestion agent
  - Final reviewer agent
- Amazon Bedrock integration behind a small `LLMClient` interface
- DynamoDB persistence behind a `Store` interface
- GitHub webhook support for `opened`, `synchronize`, and `reopened` pull request events
- HMAC validation for GitHub webhook signatures
- Optional GitHub PR comment posting
- Request validation, bounded diff size, safe error responses, structured logs, and automated tests
- AWS SAM template with API Gateway, Lambda, DynamoDB, CloudWatch logging, and scoped IAM permissions

## API

### `GET /health`

```json
{
  "status": "ok",
  "service": "prpilot"
}
```

### `POST /review`

Request:

```json
{
  "title": "Add JWT authentication middleware",
  "description": "This PR adds protected routes and token validation.",
  "diff": "diff --git a/auth.go b/auth.go ...",
  "repository": "owner/repo",
  "pull_request_url": "https://github.com/owner/repo/pull/12"
}
```

Response:

```json
{
  "review_id": "rev_abc123",
  "created_at": "2026-06-17T12:00:00Z",
  "source": "api",
  "repository": "owner/repo",
  "pull_request_url": "https://github.com/owner/repo/pull/12",
  "title": "Add JWT authentication middleware",
  "risk_level": "medium",
  "summary": "This PR adds JWT authentication middleware for protected routes.",
  "main_changes": ["Adds token validation middleware"],
  "potential_issues": ["Expired token behavior should be tested"],
  "security_notes": ["Ensure JWT signing secrets are not committed"],
  "suggested_tests": ["Test missing Authorization header"],
  "final_comment": "Good structure overall. I would add tests for expired, missing, and malformed tokens."
}
```

### `GET /reviews/{review_id}`

Fetches a stored review from DynamoDB in AWS mode or memory in local mock mode.

### `POST /webhook/github`

Receives GitHub pull request webhook events, validates the `X-Hub-Signature-256` signature when `GITHUB_WEBHOOK_SECRET` is configured, fetches the PR diff, runs the review pipeline, stores the result, and optionally posts the final review comment when `GITHUB_TOKEN` is configured.

## Local Development

The local server defaults to a mock LLM and in-memory storage, so it can run without AWS credentials.

```bash
make run
```

Try the health endpoint:

```bash
curl http://localhost:8080/health
```

Submit the sample diff:

```bash
curl -X POST http://localhost:8080/review \
  -H 'content-type: application/json' \
  -d "$(jq -Rs '{title:"Add JWT authentication middleware", diff:.}' tests/sample_diff.txt)"
```

## AWS Configuration

Required for deployed Lambda:

```text
AWS_REGION
BEDROCK_MODEL_ID
DYNAMODB_TABLE_NAME
```

Optional:

```text
GITHUB_TOKEN
GITHUB_WEBHOOK_SECRET
MAX_DIFF_CHARS
```

For local Bedrock and DynamoDB testing:

```bash
export USE_BEDROCK=true
export USE_DYNAMODB=true
export AWS_REGION=us-east-1
export BEDROCK_MODEL_ID=us.anthropic.claude-haiku-4-5-20251001-v1:0
export DYNAMODB_TABLE_NAME=prpilot_reviews
make run
```

## Deploy

Install AWS SAM, authenticate to AWS, enable access to the selected Bedrock model, then run:

```bash
make build-lambda
sam build --template infra/template.yaml
sam deploy --guided --template-file .aws-sam/build/template.yaml
```

The SAM template creates:

- API Gateway HTTP API
- Lambda function
- DynamoDB table
- CloudWatch JSON logs
- IAM permissions for DynamoDB CRUD and `bedrock:InvokeModel`

## Tests

```bash
make test
```

Coverage focuses on request validation, JSON parsing, invalid JSON retry behavior, local review handling, and GitHub webhook signature verification.

The repository also includes a GitHub Actions workflow in `.github/workflows/test.yml` that runs `go test ./...` on pull requests and pushes to `main`.

## Project Structure

```text
cmd/local      Local HTTP server
cmd/lambda     AWS Lambda entrypoint
internal/api   HTTP and Lambda handlers
internal/bedrock Bedrock client and mock LLM
internal/review Multi-stage review pipeline and prompts
internal/storage DynamoDB and memory stores
internal/github GitHub webhook and REST client
infra          AWS SAM template
tests          Sample PR diff
```

## Future Improvements

- Store GitHub credentials in AWS Secrets Manager
- Add PR review-thread comments instead of a single issue comment
- Archive raw diffs to S3 for auditability
- Add GitHub Actions CI
- Add Step Functions if the review stages need independent retries
- Add per-repository configuration for review strictness

## Resume Bullets

- Built a serverless AI pull request review assistant in Go using AWS Lambda, API Gateway, Bedrock, DynamoDB, and the GitHub API to analyze PR diffs and generate review comments automatically.
- Implemented a multi-stage review pipeline with summarizer, risk analysis, security review, and test suggestion agents to produce structured code review feedback, risk levels, issue summaries, and targeted test recommendations.
- Integrated GitHub webhooks, DynamoDB persistence, CloudWatch logging, IAM-scoped permissions, and automated Go tests to support secure PR review workflows and improve developer feedback loops.
