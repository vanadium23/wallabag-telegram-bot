# Gitea Actions Workflow

## Docker Image Publishing

This workflow automatically builds and publishes Docker images to Gitea Packages.

### Triggers

The workflow runs on:
- Pushes to `master` or `main` branch
- Version tags (e.g., `v1.0.0`, `v2.1.3`)
- Manual workflow dispatch

### Configuration

#### 1. Update Registry URL

If your Gitea instance is not hosted at `gitea.com`, update the `REGISTRY` environment variable in the workflow file:

```yaml
env:
  REGISTRY: your-gitea-instance.com  # Change this to your Gitea domain
```

#### 2. Set up Gitea Token

The workflow requires a `GITEA_TOKEN` secret with permissions to write to packages.

**To create the token:**

1. Go to your Gitea instance
2. Navigate to Settings → Applications
3. Generate a new token with `write:packages` scope
4. Add the token as a repository secret named `GITEA_TOKEN`:
   - Go to your repository settings
   - Navigate to Secrets → Actions
   - Add a new secret with name `GITEA_TOKEN` and paste your token

### Generated Docker Tags

The workflow automatically generates the following tags:

- `latest` - for default branch (master/main)
- `<branch-name>` - for branch pushes
- `<version>` - for semver tags (e.g., `1.2.3` from `v1.2.3`)
- `<major>.<minor>` - for semver tags (e.g., `1.2` from `v1.2.3`)
- `<major>` - for semver tags (e.g., `1` from `v1.2.3`)
- `<branch>-<sha>` - SHA-based tags

### Using the Published Image

After the workflow runs successfully, pull your image:

```bash
docker pull <your-gitea-instance>/<username>/wallabag-telegram-bot:latest
```

Or use a specific version:

```bash
docker pull <your-gitea-instance>/<username>/wallabag-telegram-bot:v1.0.0
```


