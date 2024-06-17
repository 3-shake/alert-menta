# alert-menta
An innovative tool 🚀 for real-time analysis and management of Issues' alerts. 🔍 It identifies alert causes, proposes actionable solutions 💡, and offers customizable filters 🎛️ and detailed reports 📈. Designed for developers 👨‍💻, managers 📋, and IT teams 💻, AlertMenta enhances productivity and software quality. 🌟

## Run Locally
```
go run ./cmd/main.go -owner <owner> -issue <issue-number> -repo <repository> -github-token $GITHUB_TOKEN -api-key $OPENAI_API_KEY -command <describe or improve> -config <User_defined_config_file>
```

## Setup to run as the GitHub Actions
### Get Tokens

1. Set up a token by following the instructions [here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).
    1. Grant access to the repository where you want to implement Alert-Menta.
    2. Set the Repository Permissions as follows:
        - Actions: Read and write
        - Content: Read and write
        - Metadata: Read-only
        - Pull requests: Read and write

### Add the secrets to your repository
Register the following environment variables as Secrets in the GitHub repository.
```
GH_TOKEN = <your-github-token>
OPENAI_API_KEY = <your-openai-api-key>
```

### Create Actions file
Create an action file in your repository as `.github/workflows/alert-menta.yaml` with the following contents.
``` yaml
name: "Alert-Menta: Reacts to specific labels"
run-name: ${{ GITHUB_REPOSITORY }} LLM responds to issues against the repository.🚀

on:
  issues:
    types: [labeled] # https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#issues
  issue_comment:
    types: [created]

jobs:
  Alert-Menta:
    if: (contains(github.event.issue.labels.*.name, '/describe') && startsWith(github.event.comment.body, '/describe')) ||
      (contains(github.event.issue.labels.*.name, '/improve') && startsWith(github.event.comment.body, '/improve')) # https://docs.github.com/ja/webhooks/webhook-events-and-payloads#issues
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
      
      - name: Set Command describe
        id: describe
        if: contains(github.event.issue.labels.*.name, '/describe')
        run: |
          echo "COMMAND=describe" >> $GITHUB_ENV
      - name: Set Command improve
        id: improve
        if: steps.describe.conclusion == 'skipped' || contains(github.event.issue.labels.*.name, '/improve')
        run: |
          echo "COMMAND=improve" >> $GITHUB_ENV
      - run: echo "REPOSITORY_NAME=${GITHUB_REPOSITORY#${GITHUB_REPOSITORY_OWNER}/}" >> $GITHUB_ENV

      - name: Get user defined config file
        id: user_config
        if: hashFiles('.alert-menta.user.yaml') != ''
        run: |
          echo "CONFIG_FILE=.alert-menta.user.yaml" >> $GITHUB_ENV
      - name: Get default config file
        if: steps.user_config.conclusion == 'skipped' && hashFiles('./internal/config/config.yaml') != ''
        run: |
          echo "CONFIG_FILE=./internal/config/config.yaml" >> $GITHUB_ENV

      - name: Add Comment
        run: |
          ./alert-menta -owner ${{ github.repository_owner }} -issue ${{ github.event.issue.number }} -repo ${{ env.REPOSITORY_NAME }} -github-token ${{ secrets.GITHUB_TOKEN }} -api-key ${{ secrets.OPENAI_API_KEY }} -command $COMMAND -config $CONFIG_FILE

```

### Create User defined config file (Optional)
If the user wishes to change the behavior or model for each command, a user-defined configuration file can be written.  
Create a `.alert-menta.user.yaml` file directly under the repository and describe the settings according to the following properties.

```yaml
system:
  debug: 
    mode: True
    log_level: debug

github:
  owner: "<owner>"
  repo: "<repository>"

ai:
  model: "gpt-3.5-turbo" # Check the list of available models by `curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY"`
  
  commands:
    - describe:
        description: "Describe the GitHub Issues"
        system_prompt: "System prompt for your configured /describe command"
    - improve:
        description: "Improve the GitHub Issues"
        system_prompt: "System prompt for your configured /describe command"
```

## Run as GitHub Actions
1. Label `/describe` or `/improve` on the relevant Issues.
2. Post a comment with the command corresponding to the added label (e.g., in an Issue with the /improve label, commenting "/improve" will fire the GitHub Actions).
3. Actions are fired and comments by LLM are posted a few seconds later.