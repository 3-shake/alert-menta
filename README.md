# alert-menta

LLM-powered incident response assistant for GitHub Issues. Reduce MTTR with AI-driven analysis, runbooks, and postmortems.

## Features

| Category | Feature | Description |
|----------|---------|-------------|
| **Commands** | `/describe` | Summarize the incident |
| | `/analysis` | Root cause analysis (5 Whys) |
| | `/suggest` | Propose improvement measures |
| | `/ask` | Answer free-text questions |
| | `/postmortem` | Generate postmortem document |
| | `/runbook` | Generate response procedures |
| | `/timeline` | Generate incident timeline |
| | `/triage` | Structured JSON triage output |
| **Providers** | OpenAI | GPT-4, GPT-4o-mini |
| | Anthropic | Claude |
| | VertexAI | Gemini |
| **Integrations** | Slack | Notification on command response |
| | MCP | Claude Code integration |
| **Automation** | First Response | Auto-post incident guide |
| | Fallback | Auto-switch providers on failure |
| | Structured Output | JSON schema-compliant responses |

## Overview

### Purpose
Reduce the burden of system failure response using LLM. Get AI-powered incident support directly within GitHub Issues.

### Main Features
- **Slash Commands**: Execute AI commands in Issue comments (`/describe`, `/analysis`, etc.)
- **Multi-Provider Support**: OpenAI, Anthropic (Claude), VertexAI (Gemini)
- **Provider Fallback**: Automatic failover between providers
- **Structured Output**: JSON responses for system integrations
- **First Response Guide**: Auto-post incident guides for new issues
- **Slack Notifications**: Get notified when AI responds
- **MCP Server**: Claude Code integration for local development
- **Image Support**: Analyze screenshots and diagrams in issues
- **Customizable Prompts**: Define your own commands and prompts
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
    model: "gpt-4o-mini" # Check the list of available models by curl https://api.openai.com/v1/models -H "Authorization: Bearer $OPENAI_API_KEY"
  vertexai:
    project: "<YOUR_PROJECT_ID>"
    location: "us-central1"
    model: "gemini-2.0-flash-001"
  commands:
    - describe:
        description: "Generate a detailed description of the Issue."
        system_prompt: "The following is the GitHub Issue and comments on it. Please Generate a detailed description.\n"
        require_intent: false
    - suggest:
        description: "Provide suggestions for improvement based on the contents of the Issue."
        system_prompt: "The following is the GitHub Issue and comments on it. Please identify the issues that need to be resolved based on the contents of the Issue and provide three suggestions for improvement.\n"
        require_intent: false
    - ask:
        description: "Answer free-text questions."
        system_prompt: "The following is the GitHub Issue and comments on it. Based on the content, provide a detailed response to the following question:\n"
        require_intent: true
```
Specify the LLM to use with `ai.provider`.
You can change the system prompt with `commands.{command}.system_prompt`.
#### Custom command
`.alert-menta.user.yaml` allows you to set up custom commands for users.
Set the following in `command.{command}`.
- `description`
- `system_prompt`: describe the primary instructions for this command.
- `require_intent`: allows the command to specify arguments. (e.g. if `require_intent` is true, we execute command that `/{command} “some instruction”`)

The built-in `analysis` command uses the 5 Whys method for root cause analysis. You can customize it or create your own RCA command:
```yaml
- analysis:
    description: "Perform root cause analysis using 5 Whys method."
    system_prompt: |
      You are an SRE expert. Perform a root cause analysis on the following incident.

      Analysis Framework:
      1. Identify the direct cause
      2. Apply 5 Whys analysis
      3. Identify the root cause
      4. List contributing factors
      5. Propose recommended actions

      Output in structured Markdown format with sections:
      - Direct Cause
      - 5 Whys Analysis
      - Root Cause
      - Contributing Factors
      - Recommended Actions
    require_intent: false
```

The built-in `postmortem` command generates comprehensive postmortem documentation from the incident Issue and its comment timeline:
```yaml
- postmortem:
    description: "Generate a postmortem document from the incident timeline."
    system_prompt: |
      You are an SRE expert. Generate a postmortem document based on the incident Issue.

      Output format:
      - Incident Summary (date, duration, severity, impact)
      - Timeline (chronological events from comments)
      - Root Cause (direct cause and contributing factors)
      - Response & Resolution
      - What Went Well / What Could Be Improved
      - Action Items (prioritized with owners)
      - Lessons Learned
    require_intent: false
```

### First Response Guide
alert-menta can automatically post an incident response guide when Issues with specific labels are created. This helps on-call responders quickly understand what steps to take.

#### Configuration
Add the following to your `.alert-menta.user.yaml`:
```yaml
first_response:
  enabled: true
  trigger_labels:
    - incident
    - alert
  guides:
    - severity: high
      auto_notify:
        - "@sre-team"
    - severity: medium
      auto_notify: []
    - severity: low
      auto_notify: []
  escalation:
    timeout_minutes: 15
    notify_target: "@oncall"
```

#### Severity Detection
Severity is automatically determined from:
1. **Labels**: `severity:high`, `sev1`, `critical`, `p0`, etc.
2. **Issue body**: Keywords like "production", "outage", "service down"

#### GitHub Actions Setup
See `.github/workflows/first-response.yaml.example` for a workflow template.

#### Local Testing
```bash
go run ./cmd/firstresponse/main.go \
  -owner <owner> -repo <repo> -issue <number> \
  -github-token $GITHUB_TOKEN \
  -config .alert-menta.user.yaml \
  -dry-run  # Preview without posting
```

### Provider Fallback
alert-menta supports automatic failover between AI providers. If the primary provider fails (timeout, rate limit, server error), it automatically tries backup providers.

#### Configuration
Add the following to your `.alert-menta.user.yaml`:
```yaml
ai:
  fallback:
    enabled: true
    providers:  # Tried in order
      - openai
      - anthropic
      # - vertexai
    retry:
      max_retries: 2    # Retries per provider
      delay_ms: 1000    # Delay between retries
```

When fallback is enabled, the primary `ai.provider` setting is ignored, and providers are tried in the order specified in `fallback.providers`.

#### Supported Providers
- `openai` - OpenAI API (GPT-4, etc.)
- `anthropic` - Anthropic API (Claude)
- `vertexai` - Google Vertex AI (Gemini)

### Structured Output
alert-menta supports structured JSON output for commands that need machine-parseable responses. This is useful for integrations with other systems.

#### Configuration
Add `structured_output` to any command in your `.alert-menta.user.yaml`:
```yaml
commands:
  - triage:
      description: "Triage incident with structured output"
      system_prompt: "Analyze the incident..."
      require_intent: false
      structured_output:
        enabled: true
        schema_name: "incident_triage"
        schema:
          type: object
          properties:
            severity:
              type: string
              enum: ["critical", "high", "medium", "low"]
            category:
              type: string
            summary:
              type: string
          required: ["severity", "category", "summary"]
        fallback_to_text: true
```

#### Output Example
```json
{
  "severity": "high",
  "category": "infrastructure",
  "summary": "API server returning 500 errors due to database connection issues"
}
```

#### Provider Support
| Provider | JSON Mode | Schema Validation |
|----------|-----------|-------------------|
| OpenAI | Yes | Yes (native) |
| Anthropic | Yes | Via prompt |
| VertexAI | Yes | Via prompt |

### Slack Notifications
alert-menta can send notifications to Slack when AI responds to commands. This is useful for keeping your team informed about incident analysis.

#### Configuration
Add the following to your `.alert-menta.user.yaml`:
```yaml
notifications:
  slack:
    enabled: true
    webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
    channel: "#incidents"  # Optional: Override webhook default channel
    notify_on:
      - command_response  # Notify when AI responds to a command
```

#### Using CLI Flag
You can also pass the webhook URL as a command-line flag:
```bash
./alert-menta -slack-webhook-url "https://hooks.slack.com/services/YOUR/WEBHOOK/URL" ...
```
The CLI flag takes precedence over the config file setting.

#### Setup Slack Webhook
1. Go to your Slack workspace settings
2. Navigate to "Apps" > "Incoming Webhooks"
3. Create a new webhook and select the channel
4. Copy the webhook URL to your config or secrets

#### GitHub Actions Integration
Add your Slack webhook URL to GitHub Secrets and update your workflow:
```yaml
- name: Add Comment
  run: |
    ./alert-menta ... -slack-webhook-url ${{ secrets.SLACK_WEBHOOK_URL }}
```

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
    if: startsWith(github.event.comment.body, '/') && (github.event.comment.author_association == 'MEMBER' || github.event.comment.author_association == 'OWNER')
    runs-on: ubuntu-24.04
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
          curl -sL "https://${{ secrets.GH_TOKEN }}@api.github.com/repos/3-shake/alert-menta/releases/tags/v0.1.2" \
          | jq '.assets[] | select(.name | contains("Linux_x86")) | .id')"
          tar -zxvf alert-menta_Linux_x86_64.tar.gz

      - run: echo "REPOSITORY_NAME=${GITHUB_REPOSITORY#${GITHUB_REPOSITORY_OWNER}/}" >> $GITHUB_ENV

      - name: Get user defined config file
        id: user_config
        if: hashFiles('.alert-menta.user.yaml') != ''
        run: |
          curl -H "Authorization: token ${{ secrets.GH_TOKEN }}" -L -o .alert-menta.user.yaml "https://raw.githubusercontent.com/${{ github.repository_owner }}/${{ env.REPOSITORY_NAME }}/main/.alert-menta.user.yaml" && echo "CONFIG_FILE=./.alert-menta.user.yaml" >> $GITHUB_ENV

      - name: Extract command and intent
        id: extract_command
        run: |
          COMMENT_BODY="${{ github.event.comment.body }}"
          COMMAND=$(echo "$COMMENT_BODY" | sed -E 's|^/([^ ]*).*|\1|')
          echo "COMMAND=$COMMAND" >> $GITHUB_ENV
          
          if [[ "$COMMENT_BODY" == "/$COMMAND "* ]]; then
            INTENT=$(echo "$COMMENT_BODY" | sed -E "s|^/$COMMAND ||")
            echo "INTENT=$INTENT" >> $GITHUB_ENV
          fi
          
          COMMANDS_CHECK=$(yq e '.ai.commands[] | keys' .alert-menta.user.yaml | grep -c "$COMMAND" || echo "0")
          if [ "$COMMANDS_CHECK" -eq "0" ]; then
            echo "Invalid command: $COMMAND. Command not found in configuration."
            exit 1
          fi

      - name: Add Comment
        run: |
          if [ -n "$INTENT" ]; then
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
## Claude Code Integration (MCP)
alert-menta provides an MCP (Model Context Protocol) server that enables Claude Code to interact with GitHub Issues directly.

### Setup
Add to your Claude Code settings (`~/.claude/settings.json`):
```json
{
  "mcpServers": {
    "alert-menta": {
      "command": "go",
      "args": ["run", "./cmd/mcp/main.go", "-config", ".alert-menta.user.yaml"],
      "cwd": "/path/to/alert-menta",
      "env": {
        "GITHUB_TOKEN": "your-github-token",
        "OPENAI_API_KEY": "your-openai-api-key"
      }
    }
  }
}
```

### Available Tools
- `get_incident`: Get incident information from a GitHub Issue
- `analyze_incident`: Run analysis commands (describe, suggest, analysis, postmortem, runbook, timeline)
- `post_comment`: Post a comment to a GitHub Issue
- `list_commands`: List all available commands

### Usage Example
```
> Get the details of Issue #123 in owner/repo
> Analyze Issue #123 using the analysis command
> Post a summary comment to Issue #123
```

## Contribution
We welcome you.
Please submit pull requests to the develop branch. See [Branch strategy](https://github.com/3-shake/alert-menta/wiki/Branch-strategy) for more information.
