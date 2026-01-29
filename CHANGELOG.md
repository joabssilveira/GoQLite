# Changelog

All notable changes to **GoQLite** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.3.0] - 2026-01-29

### ✨ Added
- Generic `JSONB[T]` type for seamless PostgreSQL JSONB support  
  - Implements `sql.Scanner` and `driver.Valuer`  
  - Compatible with GORM and `database/sql`  
  - Includes `MarshalJSON` and `UnmarshalJSON` to expose clean JSON in APIs (no wrapper field)  
  - Supports persistence of slices, structs, maps, and pointer types as JSONB  
  - Centralizes JSONB handling inside GoQLite, removing the need for per-project boilerplate

- Typed update resolver support via `UpdateStructResolver[T]`  
  - Allows update handlers to receive strongly typed payloads  
  - Brings update flow in line with the existing `GormCreateHandler` pattern  

### 🔄 Changed
- `GormUpdateHandler` now uses typed structs instead of `map[string]interface{}` by default  
  - Preserves custom field types such as:
    - `JSONB[T]`
    - custom enums
    - embedded structs  
  - Prevents type loss during updates that previously generated invalid SQL for JSONB fields  
  - Primary key protection is now enforced by restoring the persisted value before update  

### 🐛 Fixed
- Fixed PostgreSQL error when updating JSONB columns:
  - This happened because map-based updates converted JSON arrays into SQL record syntax like: ('A','B','C')

- Updates now correctly send valid JSON to PostgreSQL.

### 💥 Breaking Changes
- `GormUpdateHandler` no longer supports map-based dynamic updates  
- The previous behavior caused loss of type information and broke JSONB and other custom types  
- If dynamic/unsafe updates are required, a dedicated patch-style handler must be implemented separately

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
