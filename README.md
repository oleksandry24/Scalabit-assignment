# Scalabit Challenge

REST API built in Go to manage GitHub repositories and pull requests.
This project was built with DevSecOps practices.

## Requirements

- Create REST API that allows create, destroy, and list repositories in github
- Create REST API that allows for a certain repo list the N pull requests open
- Pipeline for running tests, lint, security check and finally deploy (in minikube)
- All done with git
- Language: go
- Testing is important

---

## How to Run

### Prerequisites

- Go 1.26+
- Docker & Minikube
- A GitHub Personal Access Token (PAT)

### 1. Environment Setup

Create a `.env` file in the root of the project and add your GitHub Token. This token needs the `repo` and `delete_repo` scopes to read, create, and destroy repositories.

```env
GITHUB_TOKEN=ghp_your_personal_access_token_here

```

### 2. Run the Application locally

```bash
go mod tidy
go run ./cmd/api/main.go

```

Dashboard will be implemented soon...

<!-- The server will start on `http://localhost:8080`. -->

---

## API Documentation

### 1. Health Check

Checks if the API is running and responding to Kubernetes probes.

- **URL:** `GET /health`
- **Success Response:** `200 OK`

### 2. List Repositories

Retrieves all repositories for the authenticated user.

- **URL:** `GET /repos`
- **Success Response:** `200 OK`
- **Response Body:** Array of GitHub Repository objects.

### 3. Create Repository

Creates a new repository in your GitHub account.

- **URL:** `POST /repos`
- **Request Body (JSON):**

```json
{
  "name": "my-new-repo"
}
```

- **Success Response:** `201 Created`
- **Response Body:** The created GitHub Repository object.

### 4. Delete (Destroy) Repository

Permanently deletes a repository.

- **URL:** `DELETE /repos/{owner}/{repo}`
- **Success Response:** `204 No Content`

### 5. List Top N Open Pull Requests

Retrieves the most recent open pull requests for a specific repository.

- **URL:** `GET /repos/{owner}/{repo}/pulls?limit=5`
- **URL Params:** `limit` (Optional, defaults to a specific number)
- **Success Response:** `200 OK`
- **Response Body:** Array of the top N open Pull Request objects.

### 6. Change Repository Visibility (Additional)

Updated the visibility of a repository.

- **URL:** `PATCH /repos/{owner}/{repo}`
- **Request Body:**

```json
{
  "private": true
}
```

- **Success Response:** `200 OK`
- **Response Body:** The updated GitHub Repository object.
  }

---

## DevSecOps & Pipeline Architecture

A core pillar of this project is a robust **Shift-Left Security** strategy. By integrating automated testing and scanning directly into the CI/CD pipeline, potential issues are caught and mitigated long before the code reaches the cluster.

### 1. Code Integrity & Early Detection

- **Golangci-Lint:** Is enforced to guarantee clean, readable code that adheres to standard Go idioms.
- **Unit Testing & Coverage Verification:** Native Go tests validate all handlers and business logic. The pipeline enforces a strict quality gate, automatically failing the build if code coverage metrics are not met (under 70%).
- **Static Application Security Testing (SAST):** Powered by `gosec` (GolangSec), the Go source code is continuously analyzed at the AST level to block insecure coding patterns, hardcoded credentials, and unsafe memory pointers.
- **Secret Scanning:** The pipeline actively scans the codebase and commit history to ensure no GitHub Tokens, API keys, or sensitive credentials are accidentally leaked into version control.

### 2. Artifact Security & SCA

- **Isolated Binaries:** Compiling with `CGO_ENABLED=0` produces a standalone, statically linked Go binary, significantly reducing the attack surface.
- **Software Composition Analysis (SCA):** Before the Docker image is allowed into the cluster, `Trivy` inspects the Alpine base image and dependencies. The deployment is instantly halted if any `CRITICAL` or `HIGH` Common Vulnerabilities and Exposures (CVEs) are detected.
- **IaC Scanning:** `Checkov` and `Trivy` act as a double-layer defense mechanism, analyzing the Kubernetes manifests (`k8s/`) to block risky misconfigurations from being applied.

### 3. Kubernetes Hardening (Runtime Security)

The application runs in a highly restricted, zero-trust Kubernetes environment:

- **Unprivileged Execution:** The container is explicitly bound to `runAsUser: 10000` and `runAsNonRoot: true`. Privilege escalation is disabled (`allowPrivilegeEscalation: false`), and all default kernel capabilities are dropped.
- **Read-Only Environment:** `readOnlyRootFilesystem: true` blocks unauthorized modifications to the disk. To maintain Go's runtime functionality, an ephemeral `emptyDir` is mounted exclusively to `/tmp`.
- **Network Isolation:** Ingress Network Policies lock down internal cluster traffic, permitting inbound requests only on the designated API port (8080).

### 4. Continuous Deployment & Smoke Testing

The CI/CD workflow provisions a temporary Minikube cluster directly on the runner. It injects the scanned image into an isolated `scalabit-challenge` namespace (`imagePullPolicy: Never`) and executes an automated smoke test against the `/health` endpoint. The pipeline only turns green if the API successfully returns an HTTP 200 response.

---

## Local Deployment Guide (Minikube)

To simulate the production environment and test the API locally, ensure Docker and Minikube are running on your machine, then follow these steps:

```bash
# 1. Initialize the cluster and create the isolated environment
minikube start --driver=docker
kubectl create namespace scalabit-challenge

# 2. Build the API and load it into Minikube's internal registry
docker build -t scalabit-challenge-api:latest .
minikube image load scalabit-challenge-api:latest

# 3. Inject the GitHub Personal Access Token (Replace "your_token_here")
kubectl create secret generic github-token \
  --from-literal=GITHUB-TOKEN="your_token_here" \
  -n scalabit-challenge

# 4. Apply the hardened infrastructure
kubectl apply -f k8s/
kubectl rollout restart deployment/scalabit-challenge-api -n scalabit-challenge

# 5. Expose the Service
minikube service scalabit-challenge-api-svc -n scalabit-challenge

```

```

```
