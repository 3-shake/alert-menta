# alert-menta
An innovative tool 🚀 for real-time analysis and management of Issues' alerts. 🔍 It identifies alert causes, proposes actionable solutions 💡, and offers customizable filters 🎛️ and detailed reports 📈. Designed for developers 👨‍💻, managers 📋, and IT teams 💻, AlertMenta enhances productivity and software quality. 🌟

## Requirements
- Go <= 1.22.2
- OpenAI API Key
- GitHub Token

## Run Locally
### Install golang
Install Go with reference to [here](https://go.dev/doc/install).  
If you are on Linux, you can follow the steps below to install the software.
```bash
$ wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
$ rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
$ echo "export PATH=\$PATH:/usr/local/go/bin" >> $HOME/.profile
```

Check the version of go. `Go 1.22.2` is used for development.
```bash
$ go version
go version go1.22.2 linux/amd64
```

### Clone alert-menta project
```bash
$ git clone https://github.com/3-shake/alert-menta.git
$ cd alert-menta
```

### Run alert-menta using go
```bash
go run ./cmd/main.go -owner <owner> -issue <issue-number> -repo <repository> -github-token $GITHUB_TOKEN -api-key $OPENAI_API_KEY -command <describe or improve> -config <User_defined_config_file>
```

## Run Locally with Docker
in the process of writing...

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
run-name: LLM responds to issues against the repository.🚀

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
  provider: "openai" # "openai" or "vertexai"
  openai:
    model: "<MODEL_ID>" # Check the list of available models by `curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY"` (e.g. gpt-3.5-turbo)

  vertexai:
    project: "<YOUR_PROJECT_ID>"
    location: "<YOUR_REGION>"
    model: "<MODEL_ID>" # e.g. gemini-1.5-flash-001
  
  commands:
    - describe:
        description: "Describe the GitHub Issues"
        system_prompt: "The following is the GitHub Issue and comments on it. Please summarize the conversation and suggest what issues need to be resolved.\n"
    - improve:
        description: "Improve the GitHub Issues"
        system_prompt: "The following is the GitHub Issue and comments on it. Please identify the issues that need to be resolved based on the contents of the Issue and provide three suggestions for improvement."
    - ask:
        description: "Ask a question about the GitHub Issue"
        system_prompt: "The following is the GitHub Issue and comments on it. Based on the content provide a detailed response to the following question:\n"
```

## Run as GitHub Actions
1. Label `/describe` or `/improve` on the relevant Issues.
2. Post a comment with the command corresponding to the added label (e.g., in an Issue with the /improve label, commenting "/improve" will fire the GitHub Actions).
3. Actions are fired and comments by LLM are posted a few seconds later.