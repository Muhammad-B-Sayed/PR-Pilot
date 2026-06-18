# PRPilot Infrastructure

This folder contains the AWS SAM template for the serverless deployment.

## Resources

- API Gateway HTTP API with `/health`, `/review`, `/reviews/{review_id}`, and `/webhook/github`
- AWS Lambda custom runtime for the Go binary
- DynamoDB table named `prpilot_reviews`
- IAM permissions scoped to the review table and selected Bedrock model
- CloudWatch JSON logs through the Lambda logging configuration

## Deploy

```bash
make build-lambda
sam build --template infra/template.yaml
sam deploy --guided --template-file .aws-sam/build/template.yaml
```

During `sam deploy --guided`, provide `GitHubToken` and `GitHubWebhookSecret` only if you want webhook-triggered PR reviews and bot comments.
