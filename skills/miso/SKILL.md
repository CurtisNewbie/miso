---
name: miso
description: "Use the miso Go framework for backend microservices. Use when working with the miso framework (https://github.com/curtisnewbie/miso) for: (1) Creating new microservices with component-based architecture, (2) Implementing RESTful APIs with Gin integration, (3) Database operations with GORM, (4) Configuration management with Viper, (5) Error handling with structured MisoErr types, (6) Distributed tracing via Rail context, (7) Bootstrap lifecycle management, (8) Service discovery and middleware integration"
---

# Miso Framework

Component-based Go microservices framework with ordered bootstrap lifecycle, distributed tracing, and pluggable middleware architecture.

## Core Concepts

miso is a Go framework for building microservices with:

- **Component Bootstrap** - Ordered component lifecycle (L1→L2→L3→L4) for clean initialization
- **Distributed Tracing** - Rail context for request tracing and structured logging
- **Web Server** - Gin integration with automatic error handling and API documentation
- **Database Middleware** - GORM integration with MySQL/SQLite support
- **Configuration Management** - Viper-based config with type-safe property constants
- **Error Handling** - Structured error types with context and stack traces
- **HTTP Client** - Service discovery and HTTP client utilities

## Quick Reference

**Core Concepts:** [core-concepts.md](references/core-concepts.md)
- Tracing, Bootstrap, Request Handling, Error Handling

**Web Development:** [web-development.md](references/web-development.md)
- Routing, API patterns, middleware, misoapi code generation

**Database:** [database.md](references/database.md)
- GORM usage, transactions, migrations, dbquery API

**Configuration:** [configuration.md](references/configuration.md)
- Viper-based config, property constants, default values

**Error Handling:** [error-handling.md](references/error-handling.md)
- Error types, wrapping, logging patterns with Rail

**Service Discovery:** [service-discovery.md](references/service-discovery.md)
- Nacos/Consul service registration, HTTP client with service discovery
