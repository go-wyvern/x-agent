FROM node:18-alpine

# Install basic dependencies (using apk for Alpine Linux)
RUN apk add --no-cache \
    curl \
    git \
    ca-certificates

# Install claude-code via npm
RUN npm install -g @anthropic-ai/claude-code

# 添加用户 (Alpine Linux 方式)
RUN addgroup -g 24368 x-agent && adduser -u 24368 -G x-agent -h /home/x-agent -s /bin/sh -D x-agent

# Create claude config directory
RUN mkdir -p /etc/claude

# Copy MCP configuration
COPY mcp-config.json /etc/claude/mcp-config.json

# Configure working directory
WORKDIR /workspace

# Change ownership of workspace and config to x-agent user
RUN chown -R x-agent:x-agent /workspace /etc/claude

# Set environment variables
ENV MCP_CONFIG_PATH="/etc/claude/mcp-config.json"

# Switch to non-root user
USER x-agent

# Startup command (keep container running)
CMD ["tail", "-f", "/dev/null"]
