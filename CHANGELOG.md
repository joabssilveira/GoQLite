# Changelog

All notable changes to **GoQLite** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.2.0] - 2026-01-28

### Changed
- **BREAKING**: `GormGetList` no longer reads `*http.Request`
- Split into:
  - `GormGetList` → core data function without HTTP dependency
  - `GormGetListHttp` → HTTP helper that parses request and calls `GormGetList`

This change separates transport (HTTP) from data querying, making the core usable in non-HTTP contexts.

---

## [0.1.0] - 2026-01-28

### Added
- Initial public release of **GoQLite**
- Core ORM-agnostic query engine
- JSON-based filtering DSL with operators:
  - `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
  - `$in`, `$nin`, `$like`, `$ilike`
  - `$between`, `$null`, `$exists`
  - Logical operators `$and`, `$or`, `$not`
- Pagination support with metadata
- Dynamic field selection
- Sorting support
- Nested relation query syntax
- Automatic relation join resolution

### GORM Adapter
- QueryBuilder implementation for GORM
- Automatic JOIN generation based on filter paths
- Nested relation preload with scoped queries
- Integration helpers for HTTP handlers
- Generic `GormGetList` with filtering, sorting, pagination and nested loading

### Documentation
- Project README with usage examples
- ROADMAP outlining future plans
- MIT License

---

## [Unreleased]

### Planned
- Expanded documentation and examples
- Unit tests for core and GORM adapter
- Integration tests using GORM DryRun
- Additional ORM adapters (Bun, Ent, SQLX)
- Performance improvements
- Security hardening for query validation
