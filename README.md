# YamlFix - YAML Fixture Library for Go

[🇯🇵 日本語](README.ja.md) | 🇺🇸 English

YamlFix is a Go library that enables using YAML files as test fixtures. It provides transaction management per test and automatic rollback functionality after tests.

## 🚀 Features

- 📁 **Load test data from YAML files** - Supports table_name.yaml format
- 🔄 **Transaction management per test** - Direct access via `GetTransaction()`
- 🔙 **Automatic rollback functionality** - Automatically cleans up data after tests
- 🗂️ **Support for related data across multiple tables** - Works with foreign key constraints
- 🧪 **Test helper functions** - Optimized for table-driven tests
- ⚡ **Simple API** - Build test environments with minimal code

## 📦 Installation

```bash
go get github.com/Yuki-TU/yamlfix
```

## 🔧 Basic Usage

### 1. Create YAML fixture files

Create files in table_name.yaml format:

```yaml
# testdata/users.yaml
- id: 1
  name: "John Doe"
  email: "john@example.com"
  created_at: "2023-01-01 10:00:00"
- id: 2
  name: "Jane Smith"
  email: "jane@example.com"
  created_at: "2023-01-02 11:00:00"
```

```yaml
# testdata/posts.yaml
- id: 1
  user_id: 1
  title: "First Post"
  content: "This is the first post"
  created_at: "2023-01-01 12:00:00"
- id: 2
  user_id: 2
  title: "Second Post"
  content: "This is the second post"
  created_at: "2023-01-02 13:00:00"
```

### 2. Basic Test

```go
package main

import (
    "database/sql"
    "testing"
    
    "github.com/Yuki-TU/yamlfix"
    _ "github.com/mattn/go-sqlite3"
)

func TestUserRepository(t *testing.T) {
    // Use SQLite in-memory database
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Initialize test fixture
    fixture := yamlfix.NewTestFixture(t, db)
    fixture.SetupTest("testdata/users.yaml", "testdata/posts.yaml")

    repo := NewUserRepository()

    // Execute with setup and test separation
    fixture.RunTestWithSetup(
        func(tx *sql.Tx) {
            // Setup phase: Create tables
            _, err := tx.Exec(`
                CREATE TABLE users (
                    id INTEGER PRIMARY KEY,
                    name TEXT NOT NULL,
                    email TEXT NOT NULL,
                    created_at TEXT
                );
                CREATE TABLE posts (
                    id INTEGER PRIMARY KEY,
                    user_id INTEGER NOT NULL,
                    title TEXT NOT NULL,
                    content TEXT,
                    created_at TEXT,
                    FOREIGN KEY (user_id) REFERENCES users(id)
                );
            `)
            if err != nil {
                t.Fatal(err)
            }
        },
        func(tx *sql.Tx) {
            // Test phase: Fixtures are automatically inserted
            users, err := repo.GetAllUsers(tx)
            if err != nil {
                t.Fatal(err)
            }

            if len(users) != 2 {
                t.Errorf("expected: 2, got: %d", len(users))
            }
        },
    )
}
```

### 3. Repository Pattern Usage

```go
type Repository struct{}

func (r *Repository) CreateUser(ctx context.Context, tx *sql.Tx, user User) (User, error) {
    query := `INSERT INTO users (name, email, created_at) VALUES (?, ?, ?)`
    result, err := tx.ExecContext(ctx, query, user.Name, user.Email, time.Now())
    if err != nil {
        return user, err
    }
    id, _ := result.LastInsertId()
    user.ID = uint64(id)
    return user, nil
}

func TestRepository(t *testing.T) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    fixture := yamlfix.NewTestFixture(t, db)
    fixture.SetupTest() // No fixture files needed for this case

    repo := NewRepository()
    ctx := context.Background()

    fixture.RunTestWithSetup(
        func(tx *sql.Tx) {
            // Create tables
            _, err := tx.Exec(`
                CREATE TABLE users (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    name TEXT NOT NULL,
                    email TEXT NOT NULL,
                    created_at DATETIME
                )
            `)
            if err != nil {
                t.Fatal(err)
            }
        },
        func(tx *sql.Tx) {
            // Table-driven tests
            tests := map[string]struct {
                user User
            }{
                "normal user creation": {
                    user: User{Name: "John Doe", Email: "john@example.com"},
                },
                "user with Japanese name": {
                    user: User{Name: "田中花子", Email: "tanaka@example.com"},
                },
            }

            for name, tt := range tests {
                t.Run(name, func(t *testing.T) {
                    created, err := repo.CreateUser(ctx, tx, tt.user)
                    if err != nil {
                        t.Fatalf("CreateUser() error = %v", err)
                    }

                    if created.ID == 0 {
                        t.Error("ID not set")
                    }
                })
            }
        },
    )
}
```

### 4. Simple Test (When tables already exist)

```go
func TestSimpleQuery(t *testing.T) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // Create tables beforehand
    _, err = db.Exec(`CREATE TABLE users (id INTEGER, name TEXT, email TEXT)`)
    if err != nil {
        t.Fatal(err)
    }

    fixture := yamlfix.NewTestFixture(t, db)
    fixture.SetupTest("testdata/users.yaml")

    // Fixtures are automatically inserted and test is executed
    fixture.RunTest(func(tx *sql.Tx) {
        var count int
        err := tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
        if err != nil {
            t.Fatal(err)
        }

        if count != 2 {
            t.Errorf("expected: 2, got: %d", count)
        }
    })
}
```

### 5. Multi-table Format (Compatibility Support)

```yaml
# testdata/multi_table.yaml
users:
  - id: 1
    name: "John Doe"
    email: "john@example.com"
    created_at: "2023-01-01 10:00:00"

posts:
  - id: 1
    user_id: 1
    title: "First Post"
    content: "This is the first post"
    created_at: "2023-01-01 12:00:00"
```

```go
func TestMultiTableFormat(t *testing.T) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    fixture := yamlfix.NewTestFixture(t, db)
    fixture.SetupTest("testdata/multi_table.yaml")

    fixture.RunTestWithSetup(
        func(tx *sql.Tx) {
            // Create tables
            _, err := tx.Exec(`
                CREATE TABLE users (id INTEGER, name TEXT, email TEXT, created_at TEXT);
                CREATE TABLE posts (id INTEGER, user_id INTEGER, title TEXT, content TEXT, created_at TEXT);
            `)
            if err != nil {
                t.Fatal(err)
            }
        },
        func(tx *sql.Tx) {
            // Fixtures are automatically inserted
            var userCount, postCount int
            tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
            tx.QueryRow("SELECT COUNT(*) FROM posts").Scan(&postCount)
            
            if userCount != 1 || postCount != 1 {
                t.Errorf("expected: users=1, posts=1, got: users=%d, posts=%d", userCount, postCount)
            }
        },
    )
}
```

## 📚 API Reference

### TestFixture (Recommended)

```go
// Create a new TestFixture instance for testing
func NewTestFixture(t *testing.T, db *sql.DB) *TestFixture

// Test setup (load YAML files)
func (tf *TestFixture) SetupTest(yamlPaths ...string)

// Execute test within transaction (automatic fixture insertion)
func (tf *TestFixture) RunTest(testFn func(tx *sql.Tx))

// Execute test after setup with fixture insertion
func (tf *TestFixture) RunTestWithSetup(setupFn func(tx *sql.Tx), testFn func(tx *sql.Tx))

// Manual control of fixture insertion timing
func (tf *TestFixture) RunTestWithCustomSetup(testFn func(tx *sql.Tx))

// Manual fixture data insertion (usually not needed)
func (tf *TestFixture) InsertTestData()

// Check if transaction is started
func (tf *TestFixture) HasTransaction() bool

// Get transaction instance (for advanced use)
func (tf *TestFixture) GetTransaction() *sql.Tx

// Test cleanup
func (tf *TestFixture) TearDownTest()
```

**Deprecated Methods (kept for compatibility)**
```go
// Deprecated: Use RunTestWithSetup or RunTest instead
func (tf *TestFixture) ExecInTransaction(query string, args ...interface{})
func (tf *TestFixture) QueryInTransaction(query string, args ...interface{}) *sql.Rows
func (tf *TestFixture) QueryRowInTransaction(query string, args ...interface{}) *sql.Row
```

## 🎯 Best Practices

### New API (Recommended)

```go
// 1. Simple case (tables already exist)
fixture.RunTest(func(tx *sql.Tx) {
    // Fixtures automatically inserted
    // Write test code only
})

// 2. Case requiring setup
fixture.RunTestWithSetup(
    func(tx *sql.Tx) {
        // Table creation and setup
    },
    func(tx *sql.Tx) {
        // Fixtures automatically inserted
        // Test code
    },
)

// 3. Case requiring complex control
fixture.RunTestWithCustomSetup(func(tx *sql.Tx) {
    // Table creation
    // Manual fixture insertion
    fixture.InsertTestData()
    // Test code
})
```

### 🆚 Old vs New API Comparison

| Feature            | Old API                             | New API          |
| ------------------ | ----------------------------------- | ---------------- |
| Fixture Insertion  | `fixture.InsertTestData()` required | Automatic        |
| Transaction Access | `fixture.GetTransaction()`          | Direct parameter |
| SQL Execution      | `fixture.ExecInTransaction()`       | `tx.Exec()`      |
| Error Handling     | Automatic in helper methods         | Explicit control |
| Readability        | Verbose                             | Concise          |
| Flexibility        | Limited                             | High             |

### 💡 Migration Guide

```go
// Old API
fixture.RunTest(func() {
    fixture.ExecInTransaction("CREATE TABLE ...")
    fixture.InsertTestData()
    rows := fixture.QueryInTransaction("SELECT ...")
})

// New API
fixture.RunTestWithSetup(
    func(tx *sql.Tx) {
        tx.Exec("CREATE TABLE ...")
    },
    func(tx *sql.Tx) {
        rows, _ := tx.Query("SELECT ...")
    },
)
```

## ⚙️ Configuration

### Recommended API (TestFixture)

When using the recommended API `NewTestFixture()`, configuration is automatically optimized:

```go
// Automatic configuration: AutoRollback = true (optimal for testing)
fixture := yamlfix.NewTestFixture(t, db)
```

### Low-level API (Fixture)

When using the low-level API `New()`, you can manually configure `Config`:

```go
// Manual configuration example 1: For testing (auto rollback enabled)
config := yamlfix.Config{
    DB:           db,
    AutoRollback: true, // Auto rollback after tests
}
fixture := yamlfix.New(config)

// Manual configuration example 2: For production (manual commit)
config := yamlfix.Config{
    DB:           db,
    AutoRollback: false, // Manual commit/rollback control
}
fixture := yamlfix.New(config)

// Low-level API usage example
err := fixture.WithTransaction(func() error {
    return fixture.InsertFixtures()
}) // Auto commit when AutoRollback=false
```

### Config Fields

```go
type Config struct {
    DB           *sql.DB // Database connection
    AutoRollback bool    // Enable automatic rollback
}
```

| Field          | Description                                                          | Recommended Setting                    |
| -------------- | -------------------------------------------------------------------- | -------------------------------------- |
| `DB`           | Database connection                                                  | Required                               |
| `AutoRollback` | `true`: Auto rollback after tests<br>`false`: Manual commit/rollback | Testing: `true`<br>Production: `false` |

**💡 Tip**: In most cases, the automatic configuration of `NewTestFixture()` is sufficient.

## 🗄️ Supported Databases

- **SQLite** (recommended for testing)
- **MySQL**
- **PostgreSQL**
- Other `database/sql` compatible databases

## 📁 Example Project Structure

```
your-project/
├── main.go
├── repository.go
├── repository_test.go
└── testdata/
    ├── users.yaml
    ├── posts.yaml
    └── categories.yaml
```

## 🤝 Contributing

Pull requests and issues are welcome!

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## 📄 License

MIT License

## 🔗 Related Links

- [Go Official Website](https://golang.org/)
- [database/sql Package](https://pkg.go.dev/database/sql)
- [YAML Specification](https://yaml.org/) 
