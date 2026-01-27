# Roadmap

This document outlines the planned evolution of **GoQLite**.

GoQLite aims to become a flexible, ORM-agnostic dynamic query engine for Go APIs, starting with GORM and expanding over time.

---

## 🎯 Short-Term Goals

### Core Stability
- [ ]  Improve and document the filter DSL
- [ ]  Validate edge cases in logical operators (`$and`, `$or`, `$not`)
- [ ]  Harden JSON parsing and error handling
- [ ]  Improve nested query parsing reliability

### GORM Adapter
- [ ]  Increase test coverage for SQL generation
- [ ]  Improve automatic join detection
- [ ]  Support deeper nested relation chains
- [ ]  Add safeguards against invalid field names

### Documentation
- [ ]  Expand README with more real-world examples
- [ ]  Add example projects under `/examples`
- [ ]  Document nested syntax clearly
- [ ]  Add “common patterns” guide

---

## 🚧 Mid-Term Goals

### Testing & Quality
- [ ]  Full unit test suite for core package
- [ ]  Integration tests using GORM DryRun mode
- [ ]  Benchmark performance for complex filters
- [ ]  Static analysis and linting setup

### Features
- [ ]  Custom operator extension support
- [ ]  Field aliasing
- [ ]  Configurable default limits and max limits
- [ ]  Soft-delete aware filtering helpers
- [ ]  JSON/JSONB advanced operators

### Developer Experience
- [ ]  Better error messages for invalid queries
- [ ]  Debug mode to print generated SQL
- [ ]  Logging hooks

---

## 🌍 Long-Term Vision

### Multi-ORM Support
- [ ]  Adapter for Bun ORM
- [ ]  Adapter for Ent
- [ ]  Adapter for SQLX
- [ ]  Generic SQL builder adapter

### Ecosystem Integrations
- [ ]  Gin middleware helpers
- [ ]  Echo middleware helpers
- [ ]  Fiber middleware helpers

### Advanced Query Capabilities
- [ ]  Aggregations (COUNT, SUM, AVG, etc.)
- [ ]  Group By support
- [ ]  Subquery support
- [ ]  Policy-based filtering (multi-tenant, RBAC)

---

## 🤝 Community Goals

- [ ]  Contribution guidelines
- [ ]  Good first issues for new contributors
- [ ]  Semantic versioning with release notes
- [ ]  Community-driven feature proposals

---

This roadmap is open to change based on community feedback and real-world usage.
