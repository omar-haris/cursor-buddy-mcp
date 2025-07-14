# System Architecture
Category: architecture
Tags: overview, design

## Components
- **API Gateway** - Single entry point
- **Auth Service** - User authentication
- **Database** - PostgreSQL with migrations
- **File Storage** - Local uploads folder

## Patterns
- Repository pattern for data access
- Event-driven communication
- Circuit breaker for external calls 