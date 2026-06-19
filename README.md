# PRPilot: AI Pull Request Review Assistant

PRPilot is a deployed serverless AI pull request review assistant. It analyzes GitHub pull request diffs, generates structured code review feedback with Amazon Bedrock, stores the review in DynamoDB, and can post a final review comment back onto the PR.

It supports both direct API review requests and real GitHub webhook-triggered reviews.

## What It Does

- Accepts a pasted diff or a real GitHub pull request webhook
- Fetches the PR diff through the GitHub API
- Runs a multi-stage AI review pipeline:
  - Summarizer
  - Risk analysis
  - Security review
  - Test suggestion
  - Final reviewer
- Returns and stores structured review JSON
- Posts the final review comment back to the GitHub PR when token permissions are configured

## Why It Matters

Code review is one of the slowest feedback loops in software teams. PRPilot demonstrates how serverless cloud architecture and LLMs can shorten that loop by automatically summarizing changes, identifying risk, highlighting security issues, and suggesting targeted tests as soon as a pull request opens or updates.

The project is intentionally small, but it uses the same patterns as larger production developer tools: webhooks, queue-backed async processing, cloud persistence, model invocation, structured logs, and scoped IAM permissions.

## Tech Stack

- **Language:** Go
- **Compute:** AWS Lambda
- **API:** API Gateway HTTP API
- **AI:** Amazon Bedrock Runtime
- **Model:** `us.anthropic.claude-haiku-4-5-20251001-v1:0`
- **Queue:** Amazon SQS
- **Database:** DynamoDB
- **Logs:** CloudWatch
- **Integration:** GitHub API and GitHub webhooks
- **Infrastructure:** AWS SAM
- **Tests:** Go test and GitHub Actions

## Architecture

```text
Manual API user
      |
      v
API Gateway HTTP API
      |
      v
Go Lambda
      |
      |----> Amazon Bedrock Runtime
      |----> DynamoDB: prpilot_reviews
      |----> CloudWatch Logs


GitHub pull request webhook
      |
      v
API Gateway HTTP API
      |
      v
Go Lambda: validate signature and enqueue
      |
      v
SQS: prpilot-github-webhooks
      |
      v
Go Lambda: process queued PR review
      |
      |----> GitHub API: fetch PR diff
      |----> Amazon Bedrock Runtime
      |----> DynamoDB: save full review
      |----> GitHub API: post PR comment
      |----> CloudWatch Logs
```

The GitHub webhook path is asynchronous because a full multi-agent Bedrock review can take longer than GitHub's webhook timeout window. The webhook Lambda returns quickly with `202`, then SQS triggers the background review.

## Demo Output

Example review generated from a real GitHub PR webhook:

```text
review_id:   rev_f0f754f8a3640aec
source:      github_webhook
repository:  Muhammad-B-Sayed/Channel-Wire
title:       Allow Vercel preview origins
risk_level:  high
```

Example final review comment:

```text
Thanks for adding Vercel preview deployment support! This is a useful feature,
but there are several concerns worth addressing:

Security & Configuration Issues:
- The CORS regex allows broad preview origins. Confirm this cannot accidentally
  match untrusted domains.
- Make sure production origins remain explicitly configured and restricted.

Testing:
- Add tests for valid Vercel preview URLs.
- Add tests for malformed preview URLs.
- Add tests confirming production origins are still enforced.
```

Stored DynamoDB review records include:

```text
summary
risk_level
main_changes
potential_issues
security_notes
suggested_tests
final_comment
```

## Current Status

The project is deployed and working as a cloud MVP:

- API Gateway endpoint is live
- Lambda handles API requests and GitHub webhooks
- SQS prevents GitHub webhook timeouts while Bedrock runs
- Bedrock generates real AI review output
- DynamoDB stores complete review records
- GitHub webhook reviews are generated from real PR diffs
- GitHub PR comments work when the token has the right permissions
- Go tests pass locally and in GitHub Actions

Deployed API:

```text
https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod
```

## Features

- Go backend with local HTTP mode and AWS Lambda mode
- Manual diff review endpoint: `POST /review`
- Stored review lookup endpoint: `GET /reviews/{review_id}`
- GitHub webhook endpoint: `POST /webhook/github`
- Multi-stage AI pipeline:
  - Summarizer agent
  - Risk analysis agent
  - Security review agent
  - Test suggestion agent
  - Final reviewer agent
- Amazon Bedrock Runtime integration behind an `LLMClient` interface
- Active Bedrock inference profile support:

```text
us.anthropic.claude-haiku-4-5-20251001-v1:0
```

- DynamoDB persistence behind a `Store` interface
- SQS-backed async GitHub webhook processing
- GitHub HMAC webhook signature validation
- GitHub PR diff fetching
- GitHub PR comment posting
- Structured CloudWatch logs
- Request validation and diff size limits
- Automated Go tests
- AWS SAM infrastructure template

## Use the Hosted API

You can use the deployed PRPilot API directly without deploying your own AWS stack.

Base URL:

```text
https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod
```

Health check:

```bash
curl https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/health
```

Submit a diff for review:

```bash
cd /Users/muhammad/Random_Projects/PR_Pilot/PR-Pilot

python3 - <<'PY' | curl -sS -X POST https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/review \
  -H 'content-type: application/json' \
  -d @-
import json
from pathlib import Path

body = {
    "title": "Add JWT authentication middleware",
    "description": "Adds protected routes and token validation.",
    "diff": Path("tests/sample_diff.txt").read_text(),
    "repository": "demo/prpilot",
    "pull_request_url": "https://github.com/demo/prpilot/pull/12",
}
print(json.dumps(body))
PY
```

Fetch a saved review:

```bash
curl https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/reviews/<review_id>
```

Hosted API note: this endpoint runs in the project owner's AWS account and uses the project owner's Bedrock, Lambda, API Gateway, SQS, and DynamoDB resources. For heavier usage, private credentials, or production usage, deploy your own stack using the instructions below.

## API

### `GET /health`

Checks whether the service is reachable.

```bash
curl https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/health
```

Response:

```json
{
  "service": "prpilot",
  "status": "ok"
}
```

### `POST /review`

Reviews a pasted diff directly.

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
  "created_at": "2026-06-18T22:03:44Z",
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

Run against the deployed API using the sample diff:

```bash
cd /Users/muhammad/Random_Projects/PR_Pilot/PR-Pilot

python3 - <<'PY' | curl -sS -X POST https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/review \
  -H 'content-type: application/json' \
  -d @-
import json
from pathlib import Path

body = {
    "title": "Add JWT authentication middleware",
    "description": "Adds protected routes and token validation.",
    "diff": Path("tests/sample_diff.txt").read_text(),
    "repository": "demo/prpilot",
    "pull_request_url": "https://github.com/demo/prpilot/pull/12",
}
print(json.dumps(body))
PY
```

### `GET /reviews/{review_id}`

Fetches a stored review.

```bash
curl https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/reviews/rev_c7910e69c2db47dd
```

### `POST /webhook/github`

Receives GitHub pull request webhook events.

Webhook URL:

```text
https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/webhook/github
```

Supported pull request actions:

```text
opened
synchronize
reopened
```

Ignored actions include:

```text
closed
```

Expected GitHub webhook response:

```json
{
  "status": "queued"
}
```

with HTTP status:

```text
202
```

The review then runs in the background through SQS.

## GitHub Webhook Setup

In the target GitHub repository:

1. Go to `Settings`.
2. Go to `Webhooks`.
3. Click `Add webhook`.
4. Set Payload URL:

```text
https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/webhook/github
```

5. Set Content type:

```text
application/json
```

The backend also supports GitHub's form-encoded `payload=...` format.

6. Set Secret to the same value deployed as `GITHUB_WEBHOOK_SECRET`.
7. Keep SSL verification enabled.
8. Choose `Let me select individual events`.
9. Select `Pull requests`.
10. Keep `Active` checked.

## GitHub Token Permissions

To post comments back onto PRs, use a GitHub fine-grained personal access token with access to the target repository.

Required repository permissions:

```text
Metadata: Read-only
Contents: Read-only
Pull requests: Read and write
Issues: Read and write
```

`Issues: Read and write` is required because GitHub pull request conversation comments are created through the Issues comments API.

Do not commit the token to the repo. Deploy it as the SAM parameter `GitHubToken`.

## Local Development

Local mode uses a mock LLM and in-memory storage by default. No AWS credentials are required.

```bash
cd /Users/muhammad/Random_Projects/PR_Pilot/PR-Pilot
make run
```

Local server:

```text
http://localhost:8080
```

Health check:

```bash
curl http://localhost:8080/health
```

Submit the sample diff locally:

```bash
python3 - <<'PY' | curl -sS -X POST http://localhost:8080/review \
  -H 'content-type: application/json' \
  -d @-
import json
from pathlib import Path

body = {
    "title": "Add JWT authentication middleware",
    "description": "Adds protected routes and token validation.",
    "diff": Path("tests/sample_diff.txt").read_text(),
    "repository": "demo/prpilot",
    "pull_request_url": "https://github.com/demo/prpilot/pull/12",
}
print(json.dumps(body))
PY
```

## AWS Configuration

Required for deployed Lambda:

```text
BEDROCK_MODEL_ID
DYNAMODB_TABLE_NAME
GITHUB_WEBHOOK_QUEUE_URL
```

Optional but used for GitHub integration:

```text
GITHUB_WEBHOOK_SECRET
GITHUB_TOKEN
MAX_DIFF_CHARS
```

Default Bedrock model/inference profile:

```text
us.anthropic.claude-haiku-4-5-20251001-v1:0
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

## Deploy Your Own AWS Stack

Prerequisites:

- Go 1.22 or newer
- AWS CLI
- AWS SAM CLI
- AWS credentials configured
- Access to the selected Bedrock inference profile

Build and deploy:

```bash
cd /Users/muhammad/Random_Projects/PR_Pilot/PR-Pilot

sam build --template infra/template.yaml

sam deploy --guided --template-file .aws-sam/build/template.yaml
```

Recommended stack values:

```text
Stack name: prpilot
Region: us-east-1
BedrockModelId: us.anthropic.claude-haiku-4-5-20251001-v1:0
```

For non-interactive deploys:

```bash
sam build --template infra/template.yaml

sam deploy \
  --template-file .aws-sam/build/template.yaml \
  --stack-name prpilot \
  --region us-east-1 \
  --capabilities CAPABILITY_IAM \
  --resolve-s3 \
  --no-confirm-changeset \
  --no-fail-on-empty-changeset \
  --parameter-overrides \
    ParameterKey=BedrockModelId,ParameterValue=us.anthropic.claude-haiku-4-5-20251001-v1:0 \
    ParameterKey=GitHubWebhookSecret,ParameterValue="$GITHUB_WEBHOOK_SECRET" \
    ParameterKey=GitHubToken,ParameterValue="$GITHUB_TOKEN"
```

The SAM template creates:

- API Gateway HTTP API
- Lambda function
- SQS queue for GitHub webhook processing
- DynamoDB table
- CloudWatch JSON logs
- IAM permissions for DynamoDB, SQS, and Bedrock invoke access

## Tests

Run all tests:

```bash
make test
```

The tests cover:

- Request validation
- JSON parsing
- Invalid JSON retry behavior
- Local review handling
- GitHub webhook signature verification
- GitHub form-encoded webhook payload parsing

GitHub Actions also runs:

```bash
go test ./...
```

on pushes to `main` and pull requests.

## Useful AWS Commands

Check deployed API:

```bash
curl https://29zk7kfu3f.execute-api.us-east-1.amazonaws.com/prod/health
```

Tail Lambda logs:

```bash
aws logs tail /aws/lambda/prpilot-PRPilotFunction-W1sWUhHxYFgN \
  --region us-east-1 \
  --since 30m \
  --format short
```

List latest DynamoDB reviews:

```bash
aws dynamodb scan \
  --table-name prpilot_reviews \
  --region us-east-1 \
  --projection-expression 'review_id, created_at, repository, title, #src, risk_level' \
  --expression-attribute-names '{"#src":"source"}' \
  --output table
```

## Project Structure

```text
cmd/local          Local HTTP server
cmd/lambda         AWS Lambda entrypoint for API Gateway and SQS events
internal/api       HTTP and Lambda handlers
internal/bedrock   Bedrock client and mock LLM
internal/config    Environment configuration
internal/github    GitHub webhook parsing and REST client
internal/queue     SQS queue adapter
internal/review    Multi-stage review pipeline and prompts
internal/storage   DynamoDB and memory stores
infra              AWS SAM template
tests              Sample PR diff
```

## Production Hardening

This is complete as a deployed portfolio MVP. Before using it as a real public production service, add:

- API authentication for `/review`
- AWS Secrets Manager for GitHub token and webhook secret
- CloudWatch alarms for Lambda errors, SQS age, and Bedrock failures
- AWS Budgets alerts for Bedrock/API usage
- Idempotency for repeated GitHub webhook deliveries
- A dead-letter queue for failed SQS review jobs
- Per-repository configuration
- WAF or stricter API Gateway throttling

## Resume Bullets

- Built a serverless AI pull request review assistant in Go using AWS Lambda, API Gateway, SQS, Bedrock, DynamoDB, and the GitHub API to analyze PR diffs and generate review comments automatically.
- Implemented a multi-stage review pipeline with summarizer, risk analysis, security review, and test suggestion agents to produce structured code review feedback, risk levels, issue summaries, and targeted test recommendations.
- Integrated GitHub webhooks, asynchronous SQS processing, DynamoDB persistence, CloudWatch logging, IAM-scoped permissions, GitHub PR comments, and automated Go tests to support secure PR review workflows and improve developer feedback loops.
