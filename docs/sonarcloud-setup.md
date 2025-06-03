# SonarCloud Integration Setup

This guide explains how to set up SonarCloud integration for the Nebengjek project to analyze code quality and coverage.

## Prerequisites

1. **SonarCloud Account**: Create an account at [sonarcloud.io](https://sonarcloud.io)
2. **GitHub Integration**: Connect your GitHub account to SonarCloud
3. **Organization Setup**: Create or join an organization in SonarCloud

## Setup Steps

### 1. Create SonarCloud Project

1. Log in to SonarCloud
2. Click "Create new project"
3. Select your GitHub repository
4. Choose your organization
5. Set the project key (should match `sonar.projectKey` in `sonar-project.properties`)

### 2. Configure Repository Secrets

Add the following secrets to your GitHub repository:

- `SONAR_TOKEN`: Generate this token in SonarCloud under "My Account" > "Security" > "Generate Tokens"

### 3. Update SonarCloud Configuration

Edit `sonar-project.properties` and update:

```properties
sonar.projectKey=your-project-key
sonar.organization=your-organization-key
```

Replace:
- `your-project-key`: Use the project key from SonarCloud (usually `owner_repository-name`)
- `your-organization-key`: Use your SonarCloud organization key

### 4. Workflow Integration

The workflows are already configured to:

1. Run Go tests with coverage
2. Convert Go coverage to XML format (required by SonarCloud)
3. Upload results to SonarCloud for analysis

## What Gets Analyzed

- **Code Quality**: Code smells, bugs, vulnerabilities
- **Test Coverage**: Line and branch coverage from Go tests
- **Security**: Security hotspots and vulnerabilities
- **Maintainability**: Technical debt and code complexity

## Coverage Reports

Coverage data is automatically:
1. Generated during `go test` execution
2. Converted to XML format using `gocov` and `gocov-xml`
3. Uploaded to SonarCloud for analysis
4. Displayed in SonarCloud dashboard and PR comments

## Viewing Results

- **SonarCloud Dashboard**: View detailed analysis at sonarcloud.io
- **GitHub PR Comments**: SonarCloud will comment on PRs with quality gate status
- **GitHub Actions**: Build status shows if quality gate passes

## Quality Gate

Configure quality gate conditions in SonarCloud:
- Coverage threshold (e.g., minimum 80%)
- No new bugs or vulnerabilities
- Technical debt ratio limits

## Exclusions

The following are excluded from analysis (configured in `sonar-project.properties`):
- Test files (`**/*_test.go`)
- Vendor dependencies
- Build artifacts (`**/bin/**`)
- Documentation (`**/docs/**`, `**/*.md`)
- Database migrations (`**/migrations/**`)
