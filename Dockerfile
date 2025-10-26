FROM ubuntu:22.04

# Install basic dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install claude-code
RUN curl -fsSL https://claude.ai/install.sh | sh

# Configure working directory
WORKDIR /workspace

# Copy MCP configuration template
COPY mcp-config.json /etc/claude/mcp-config.json

# Set environment variables
ENV CLAUDE_API_KEY=""
ENV MCP_CONFIG_PATH="/etc/claude/mcp-config.json"

# Startup command (keep container running)
CMD ["tail", "-f", "/dev/null"]
