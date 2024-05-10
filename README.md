# alert-menta
An innovative tool 🚀 for real-time analysis and management of Issues' alerts. 🔍 It identifies alert causes, proposes actionable solutions 💡, and offers customizable filters 🎛️ and detailed reports 📈. Designed for developers 👨‍💻, managers 📋, and IT teams 💻, AlertMenta enhances productivity and software quality. 🌟

## Run Locally
```
go run ./cmd/main.go -owner "repository_owner" -issue 1 -repo "repository_name" -token $GITHUB_TOKEN -comment "Comment Body"
```

## Setup
### Get Tokens

1. Set up a token by following the instructions [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).
    1. Grant access to the repository where you want to implement Alert-Menta.
    2. Set the Repository Permissions as follows:
        - Actions: Read and write
        - Content: Read and write
        - Metadata: Read-only
        - Pull requests: Read and write

### Run as the GitHub Actions
1. Add the secrets to your repository
```
GH_TOKEN = <your-github-token>
```

2. Create actions file as `.github/workflows/alert-menta.yaml` in your own repository.
``` yaml
name: Reacts to specific labels
run-name: ${{ github.actor }} is testing out GitHub Actions 🚀

on:
  issues:
    types: [labeled]

jobs:
  Alert-Menta:
    if: contains(github.event.issue.labels.*.name, '/describe')
    runs-on: ubuntu-22.04
    permissions:
      issues: write
      contents: read
    steps:
      - name: Check out repository code
        uses: actions/checkout@v4

      - name: Download and Install alert-menta
        run: |
          curl -sLJO -H 'Accept: application/octet-stream' \
          "https://${{ secrets.GH_TOKEN }}@api.github.com/repos/3-shake/alert-menta/releases/assets/$( \
          curl -sL "https://${{ secrets.GH_TOKEN }}@api.github.com/repos/3-shake/alert-menta/releases/tags/v0.0.1" \
          | jq '.assets[] | select(.name | contains("Linux_x86")) | .id')"
          tar -zxvf alert-menta_Linux_x86_64.tar.gz
      
      - run: echo "REPOSITORY_NAME=${GITHUB_REPOSITORY#${GITHUB_REPOSITORY_OWNER}/}" >> $GITHUB_ENV
      - name: Add Comment
        run: |
          ./alert-menta -owner ${{ github.repository_owner }} -issue ${{ github.event.issue.number }} -repo ${{ env.REPOSITORY_NAME }} -token ${{ secrets.GITHUB_TOKEN }} -comment "Body: $BODY"
        env:
          BODY: >
            This is test comment.  
            repository is ${{ env.REPOSITORY_NAME }}  
            runner.os is ${{ runner.os }}  
            title is ${{ github.event.issue.title }}  
            body is \"${{ github.event.issue.body }}\"

```

3. Label `/describe` on the relevant Issues.