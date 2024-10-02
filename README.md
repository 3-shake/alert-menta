# alert-menta
An innovative tool 🚀 for real-time analysis and management of Issues' alerts. 🔍 It identifies alert causes, proposes actionable solutions, 💡and detailed reports. 📈
Designed for developers 👨‍💻, managers 📋, and IT teams .💻 Alert-menta enhances productivity and software quality. 🌟

## Overview of alert-menta
### The purpose of alert-menta
We reduce the burden of system failure response using LLM.
### Main Features
You can receive support for failure handling that is completed within GitHub.
- Execute commands interactively in GitHub Issue comments:
  - `describe` command to summarize the Issue
  - `analysis` command for root cause analysis of failures (in development)
  - `suggest` command for proposing improvement measures for failures
  - `ask` command for asking additional questions
- Mechanism to improve response accuracy using [RAG](https://cloud.google.com/use-cases/retrieval-augmented-generation?hl=en) (in development)
- Selectable LLM models (OpenAI, VertexAI)
- Extensible prompt text
  - Multilingual support

## How to Use
Alert-menta is intended to be run on GitHub Actions.
### 1. Prepare GitHub PAT
Prepare a GitHub PAT with the following permissions and register it in Secrets:
- repo
- workflow
### 2. Configure to use LLM
#### Open AI
Generate an API key and register it in Secrets.
#### Vertex AI
Enable Vertex AI on Google Cloud.
Alert-menta obtains access to VertexAI using [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation). Please see [here](#if-using-vertex-ai) for details.
### 3. Create the alert-menta configuration file
Create the alert-menta configuration file in the root of the repository. For details, please see [here](#alert-mentauseryaml).
### 4. Create the Actions configuration file
There is a [template](#template) available, so please use it.
### 5. Monitoring alerts or user reports are received on Issues
For the method to bring monitoring alerts to Issues, please see [this repository](https://github.com/kechigon/alert-menta-lab/tree/main).
### 6. Execute alert-menta
Execute commands on the Issue. Run commands with a backslash at the beginning (e.g., `/describe`). For the `ask` command, leave a space and enter the question (e.g., `/ask What about the Next Action?`). Alert-menta includes the text of the Issue in the prompt and sends it to the LLM, then posts the response as a comment on the Issue.

## Configuration
### .alert-menta.user.yaml
It contains information such as the LLM model to use, system prompt text for each command, etc. The `.alert-menta.user.yaml` in this repository is a template. The contents are as follows:
```
system:
  debug:
    log_level: debug
ai:
  provider: "openai" # "openai" or "vertexai"
  openai:
    model: "gpt-3.5-turbo" # Check the list of available models by curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY"
  vertexai:
    project: "<YOUR_PROJECT_ID>"
    location: "us-central1"
    model: "gemini-1.5-flash-001"
  commands:
    - describe:
        description: "Generate a detailed description of the Issue."
        system_prompt: "The following is the GitHub Issue and comments on it. Please Generate a detailed description.\n"
    - suggest:
        description: "Provide suggestions for improvement based on the contents of the Issue."
        system_prompt: "The following is the GitHub Issue and comments on it. Please identify the issues that need to be resolved based on the contents of the Issue and provide three suggestions for improvement.\n"
    - ask:
        description: "Answer free-text questions."
        system_prompt: "The following is the GitHub Issue and comments on it. Based on the content, provide a detailed response to the following question:\n"
```
Specify the LLM to use with `ai.provider`.
You can change the system prompt with `commands.{command}.system_prompt`.
### Actions
#### Template
The `.github/workflows/alert-menta.yaml` in this repository is a template. The contents are as follows:
```
name: "Alert-Menta: Reacts to specific commands"
run-name: LLM responds to issues against the repository.🚀

on:
  issue_comment:
    types: [created]

jobs:
  Alert-Menta:
    if: (startsWith(github.event.comment.body, '/describe') || startsWith(github.event.comment.body, '/suggest') || startsWith(github.event.comment.body, '/ask')) && (github.event.comment.author_association == 'MEMBER' || github.event.comment.author_association == 'OWNER')
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
          curl -sL "https://${{ secrets.GH_TOKEN }}@api.github.com/repos/3-shake/alert-menta/releases/tags/v0.1.0" \
          | jq '.assets[] | select(.name | contains("Linux_x86")) | .id')"
          tar -zxvf alert-menta_Linux_x86_64.tar.gz

      - name: Set Command
        id: set_command
        run: |
          COMMENT_BODY="${{ github.event.comment.body }}"
          if [[ "$COMMENT_BODY" == /ask* ]]; then
            COMMAND=ask
            INTENT=${COMMENT_BODY:5}
            echo "INTENT=$INTENT" >> $GITHUB_ENV
          elif [[ "$COMMENT_BODY" == /describe* ]]; then
            COMMAND=describe
          elif [[ "$COMMENT_BODY" == /suggest* ]]; then
            COMMAND=suggest
          fi
          echo "COMMAND=$COMMAND" >> $GITHUB_ENV

      - run: echo "REPOSITORY_NAME=${GITHUB_REPOSITORY#${GITHUB_REPOSITORY_OWNER}/}" >> $GITHUB_ENV

      - name: Get user defined config file
        id: user_config
        if: hashFiles('.alert-menta.user.yaml') != ''
        run: |
          curl -H "Authorization: token ${{ secrets.GH_TOKEN }}" -L -o .alert-menta.user.yaml "https://raw.githubusercontent.com/${{ github.repository_owner }}/${{ env.REPOSITORY_NAME }}/main/.alert-menta.user.yaml" && echo "CONFIG_FILE=./.alert-menta.user.yaml" >> $GITHUB_ENV

      - name: Add Comment
        run: |
          if [[ "$COMMAND" == "ask" ]]; then
            ./alert-menta -owner ${{ github.repository_owner }} -issue ${{ github.event.issue.number }} -repo ${{ env.REPOSITORY_NAME }} -github-token ${{ secrets.GH_TOKEN }} -api-key ${{ secrets.OPENAI_API_KEY }} -command $COMMAND -config $CONFIG_FILE -intent "$INTENT"
          else
            ./alert-menta -owner ${{ github.repository_owner }} -issue ${{ github.event.issue.number }} -repo ${{ env.REPOSITORY_NAME }} -github-token ${{ secrets.GH_TOKEN }} -api-key ${{ secrets.OPENAI_API_KEY }} -command $COMMAND -config $CONFIG_FILE
          fi
```
#### If using Vertex AI
Configure Workload Identity Federation with reference to the [documentation](https://cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines).
## Local
In an environment where Golang can be executed, clone the repository and run it as follows:
```
go run ./cmd/main.go -repo <repository> -owner <owner> -issue <issue-number> -github-token $GITHUB_TOKEN -api-key $OPENAI_API_KEY -command <describe, etc.> -config <User_defined_config_file>
```
## Contribution
We welcome you.
Please submit pull requests to the develop branch.
