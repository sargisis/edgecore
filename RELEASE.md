# How to Create a Release

EdgeCore uses GitHub Actions to automatically build and publish binaries when you create a new tag.

## Steps:

### 1. Create a Git Tag
```bash
# Make sure you're on main and up to date
git checkout main
git pull

# Create a new tag (use semantic versioning: v1.0.0, v1.1.0, etc.)
git tag v1.0.0

# Push the tag to GitHub
git push origin v1.0.0
```

### 2. Wait for GitHub Actions
GitHub will automatically:
- Build binaries for Linux (amd64, arm64)
- Build binaries for macOS (Intel, M1/M2)
- Build binaries for Windows (amd64)
- Create a new Release on GitHub
- Attach all binaries to the release

Check progress at: https://github.com/sargisis/edgecore/actions

### 3. Verify the Release
Once the workflow completes, visit:
https://github.com/sargisis/edgecore/releases

You should see your new release with all the binaries attached!

---

## Example: First Release

```bash
# Tag the current version as v1.0.0
git tag -a v1.0.0 -m "Initial release: Production-ready Load Balancer"
git push origin v1.0.0
```

After this, users can download with:
```bash
curl -L https://github.com/sargisis/edgecore/releases/download/v1.0.0/edgecore-linux-amd64 -o edgecore
chmod +x edgecore
```

---

## Updating README

After the first successful release, update `README.md` to remove _(coming soon)_ from Option A!
