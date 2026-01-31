# GoQLite

**GoQLite** is a dynamic query engine for REST APIs built with Go.

It provides a flexible, JSON-based query language that supports advanced filtering, sorting, pagination, field selection, and nested relations — with an initial adapter for **GORM** and an architecture designed to support other ORMs in the future.

Think of it as **GraphQL-like querying power for traditional REST APIs**.

---

## ✨ Features

- 🔎 **Rich filtering DSL**
  - `$eq`, `$ne`, `$gt`, `$gte`, `$lt`, `$lte`
  - `$in`, `$nin`
  - `$like`, `$ilike`
  - `$between`
  - `$null`, `$exists`
  - Logical operators: `$and`, `$or`, `$not`

- 🔗 **Automatic relation joins**
  - Filter by related model fields without writing manual joins

- 🌳 **Nested relation loading with query support**
  - Preload relations with their own filters, sorting, and field selection

- 📄 **Dynamic field selection**
- 📊 **Pagination with metadata**
- 🔌 **ORM-agnostic core (GORM adapter included)**
- 🧠 Designed for **generic CRUD APIs**, admin panels, dashboards, and SaaS backends

---

## 🚀 Installation

```bash
go get github.com/joabssilveira/GoQLite
```

---

## 🧩 Example Use Case

### Request

```http
GET /users?where={
  "age":{"$gte":18},
  "orders.total":{"$gt":100}
}&select=["id","name"]&sort=[{"field":"name","dir":"asc"}]
```

### What GoQLite Does

- Joins the `orders` relation automatically  
- Filters users where:
  - `age >= 18`
  - `orders.total > 100`
- Selects only `id` and `name`
- Sorts by `name ASC`

All of that without writing custom query logic.

---

## 🏗 Basic Example (GORM)

```go
db := gorm.Open(...)

r := http.Request{} // incoming request with query params

result, err := goqlite.GormGetListHttp[User](db, r, goqlite.Filter{})
if err != nil {
    log.Fatal(err)
}

fmt.Println(result.Payload)      // data
fmt.Println(result.Pagination)   // pagination metadata
```

---

## 🔍 Filter Operators

| Operator   | Description              | SQL Equivalent        |
|-----------|--------------------------|------------------------|
| `$eq`     | Equal                    | `=`                    |
| `$ne`     | Not equal                | `<>`                   |
| `$gt`     | Greater than             | `>`                    |
| `$gte`    | Greater or equal         | `>=`                   |
| `$lt`     | Less than                | `<`                    |
| `$lte`    | Less or equal            | `<=`                   |
| `$in`     | In array                 | `IN (...)`             |
| `$nin`    | Not in array             | `NOT IN (...)`         |
| `$like`   | Case-sensitive contains  | `LIKE %value%`         |
| `$ilike`  | Case-insensitive contains| `ILIKE %value%`        |
| `$between`| Range                    | `BETWEEN a AND b`      |
| `$null`   | Null check               | `IS NULL / IS NOT NULL`|
| `$exists` | Field existence (JSONB)  | `IS NULL / IS NOT NULL`|

Logical operators:

```json
{
  "$and": [ {...}, {...} ],
  "$or":  [ {...}, {...} ],
  "$not": { ... }
}
```

---

## 🌳 Nested Relations

GoQLite supports nested relation loading with independent query options.

### Example

```
nested={
  orders{
    {"where":{"status":{"$eq":"paid"}}},
    items{
      {"select":["id","name"]}
    }
  }
}
```

This will:

- Preload `orders` where status is `"paid"`
- Inside each order, preload `items` selecting only `id` and `name`

---

## 📦 Query Parameters Supported

| Param   | Purpose |
|--------|---------|
| `where`  | Filtering rules |
| `select` | Fields to return |
| `sort`   | Sorting rules |
| `limit`  | Max rows |
| `skip`   | Offset |
| `page`   | Page number (auto converts to skip) |
| `nested` | Nested relations |

All parameters use JSON format.

---

## 🧱 Architecture

GoQLite is split into two main layers:

### Core (ORM-agnostic)
- Query DSL
- Filter parsing
- Payload structures
- QueryBuilder interface

### ORM Adapter (GORM for now)
- SQL translation
- Relation resolution
- Automatic joins
- Preload handling

Future adapters may support:
- Bun
- Ent
- SQLX
- Raw SQL builders

---

## 🛣 Roadmap

- [ ]  Improve documentation
- [ ]  Add unit tests
- [ ]  Add integration tests with GORM DryRun
- [ ]  Add support for additional ORMs
- [ ]  Improve nested syntax
- [ ]  Performance benchmarks
- [ ]  Security review (SQL injection hardening)

---

## 🤝 Contributing

Contributions are welcome!

If you'd like to help:

    1. Open an issue to discuss your idea
    2. Fork the repository
    3. Create a feature branch
    4. Submit a Pull Request

Please keep the code consistent with existing patterns and include tests where possible.

---

## 📄 License

This project is open source and available under the **MIT License**.

---

## 💡 Inspiration

GoQLite brings ideas from:
- MongoDB query syntax
- Strapi REST filtering
- GraphQL-style nested querying

…into a lightweight and idiomatic Go library for REST APIs.
