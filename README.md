# Cursor Buddy MCP

MCP server that gives AI agents access to your project context: rules, knowledge, todos, database schema, and history.

## Architecture

This server implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) using the Go SDK from [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go). The server communicates over stdin/stdout using JSON-RPC 2.0, making it compatible with MCP clients like Claude Desktop.

### MCP Features Implemented

- **Tools**: 6 interactive tools for managing project context
- **Resources**: Project context resource with complete project state
- **Stdio Transport**: Standard input/output communication
- **Real-time Updates**: File monitoring with automatic reloading

## Quick Start

### 1. Pull from GitHub Registry
```bash
docker pull ghcr.io/openseawave/buddy-mcp:latest
```

### 2. Configure Cursor
Add to `.cursor/mcp.json`:
```json
{
  "mcpServers": {
    "cursor-buddy-mcp": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "/path/to/your/project/.buddy:/home/buddy/.buddy",
        "-e", "BUDDY_PATH=/home/buddy/.buddy",
        "ghcr.io/openseawave/buddy-mcp:latest"
      ]
    }
  }
}
```

### 3. Create .buddy folder
```bash
mkdir -p .buddy/{rules,knowledge,todos,database,history,backups}
```

### 4. Add content
Create files in `.buddy/` folders:
- `rules/` - Coding standards (markdown)
- `knowledge/` - Project documentation (markdown)  
- `todos/` - Task lists (markdown with checkboxes)
- `database/` - Schema files (SQL)

## Available Tools

- **buddy_get_rules** - Get coding standards
- **buddy_search_knowledge** - Search documentation
- **buddy_manage_todos** - List/update tasks
- **buddy_get_database_info** - Get schema info
- **buddy_history** - Track changes
- **buddy_backup** - Backup files

## Usage

Ask AI questions like:
- "What are our coding standards?"
- "Show me current todos"
- "Search for authentication info"

## File Format Examples

**Rules** (`.buddy/rules/style.md`):
```markdown
# Code Style
- category: development
- priority: critical

## Rules
- Use meaningful variable names
- Handle all errors
```

**Knowledge** (`.buddy/knowledge/api.md`):
```markdown
# API Documentation
- category: architecture
- tags: api, rest

## Endpoints
- GET /users - List users
```

**Todos** (`.buddy/todos/features.md`):
```markdown
# Features
- [ ] Add authentication
- [x] Setup database
- [ ] Create API endpoints
```

That's it! Restart Cursor and start asking questions about your project. 