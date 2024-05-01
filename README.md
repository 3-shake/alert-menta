# alert-menta
An innovative tool 🚀 for real-time analysis and management of Issues' alerts. 🔍 It identifies alert causes, proposes actionable solutions 💡, and offers customizable filters 🎛️ and detailed reports 📈. Designed for developers 👨‍💻, managers 📋, and IT teams 💻, AlertMenta enhances productivity and software quality. 🌟

## Run Locally
```
go run ./cmd/main.go -owner "repository_owner" -issue 1 -repo "repository_name" -token $GITHUB_TOKEN -comment "Comment Body"
```

## Setup
Create actions file as `.github/workflows/alert-menta.yaml`.
``` yaml
name: Reacts to specific labels
run-name: ${{ github.actor }} is testing out GitHub Actions 🚀

on:
  issues:
    types: [labeled] # https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#issues

jobs:
  Alert-Menta:
    if: contains(github.event.issue.labels.*.name, '/describe') # https://docs.github.com/ja/webhooks/webhook-events-and-payloads#issues
    runs-on: ubuntu-22.04 # https://docs.github.com/ja/actions/using-jobs/choosing-the-runner-for-a-job
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
        env: # https://docs.github.com/ja/actions/learn-github-actions/variables#defining-environment-variables-for-a-single-workflow
          BODY: >
            This is test comment.  
            repository is ${{ env.REPOSITORY_NAME }}  
            runner.os is ${{ runner.os }}  
            title is ${{ github.event.issue.title }}  
            body is \"${{ github.event.issue.body }}\"

```