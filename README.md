# YamlFix - Go言語用YAMLフィクスチャライブラリ

YamlFixは、Go言語でYAMLファイルをテストフィクスチャとして利用できるライブラリです。テスト単位でトランザクション管理し、テスト後の自動ロールバック機能を提供します。

## 🚀 特徴

- 📁 **YAMLファイルからテストデータを読み込み** - テーブル名.yaml形式をサポート
- 🔄 **テスト単位でのトランザクション管理** - `GetTransaction()`で直接アクセス可能
- 🔙 **自動ロールバック機能** - テスト後に自動的にデータをクリーンアップ
- 🗂️ **複数テーブルの関連データ対応** - 外部キー制約にも対応
- 🧪 **テスト用ヘルパー関数** - テーブルドリブンテストに最適
- ⚡ **シンプルなAPI** - 最小限のコードでテスト環境を構築

## 📦 インストール

```bash
go get github.com/Yuki-TU/yamlfix
```

## 🔧 基本的な使用方法

### 1. YAMLフィクスチャファイルの作成

テーブル名.yaml形式でファイルを作成します：

```yaml
# testdata/users.yaml
- id: 1
  name: "山田太郎"
  email: "yamada@example.com"
  created_at: "2023-01-01 10:00:00"
- id: 2
  name: "田中花子"
  email: "tanaka@example.com"
  created_at: "2023-01-02 11:00:00"
```

```yaml
# testdata/posts.yaml
- id: 1
  user_id: 1
  title: "最初の投稿"
  content: "これは最初の投稿です"
  created_at: "2023-01-01 12:00:00"
- id: 2
  user_id: 2
  title: "二番目の投稿"
  content: "これは二番目の投稿です"
  created_at: "2023-01-02 13:00:00"
```

### 2. 基本的なテスト

```go
package main

import (
    "database/sql"
    "testing"
    
    "github.com/Yuki-TU/yamlfix"
    _ "github.com/mattn/go-sqlite3"
)

func TestUserRepository(t *testing.T) {
    // SQLiteのメモリデータベースを使用
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // テストフィクスチャの初期化
    fixture := yamlfix.NewTestFixture(t, db)
    fixture.SetupTest("testdata/users.yaml", "testdata/posts.yaml")
    defer fixture.TearDownTest()

    repo := NewUserRepository()

    // セットアップとテストを分離した実行
    fixture.RunTestWithSetup(
        func(tx *sql.Tx) {
            // セットアップ段階：テーブル作成
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
            // テスト段階：フィクスチャは自動挿入済み
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

### 3. リポジトリパターンでの使用

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
    fixture.SetupTest() // フィクスチャファイルが不要な場合
    defer fixture.TearDownTest()

    repo := NewRepository()
    ctx := context.Background()

    fixture.RunTestWithSetup(
        func(tx *sql.Tx) {
            // テーブル作成
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
            // テーブルドリブンテスト
            tests := []struct {
                name string
                user User
            }{
                {
                    name: "正常なユーザー作成",
                    user: User{Name: "山田太郎", Email: "yamada@example.com"},
                },
                {
                    name: "日本語名のユーザー作成",
                    user: User{Name: "田中花子", Email: "tanaka@example.com"},
                },
            }

            for _, tt := range tests {
                t.Run(tt.name, func(t *testing.T) {
                    created, err := repo.CreateUser(ctx, tx, tt.user)
                    if err != nil {
                        t.Fatalf("CreateUser() error = %v", err)
                    }

                    if created.ID == 0 {
                        t.Error("IDが設定されていません")
                    }
                })
            }
        },
    )
}
```

### 4. シンプルなテスト（テーブル既存の場合）

```go
func TestSimpleQuery(t *testing.T) {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer db.Close()

    // 事前にテーブルを作成済みの場合
    _, err = db.Exec(`CREATE TABLE users (id INTEGER, name TEXT, email TEXT)`)
    if err != nil {
        t.Fatal(err)
    }

    fixture := yamlfix.NewTestFixture(t, db)
    fixture.SetupTest("testdata/users.yaml")
    defer fixture.TearDownTest()

    // フィクスチャが自動挿入されてテスト実行
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

### 5. 複数テーブル形式（互換性サポート）

```yaml
# testdata/multi_table.yaml
users:
  - id: 1
    name: "山田太郎"
    email: "yamada@example.com"
    created_at: "2023-01-01 10:00:00"

posts:
  - id: 1
    user_id: 1
    title: "最初の投稿"
    content: "これは最初の投稿です"
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
    defer fixture.TearDownTest()

    fixture.RunTestWithSetup(
        func(tx *sql.Tx) {
            // テーブル作成
            _, err := tx.Exec(`
                CREATE TABLE users (id INTEGER, name TEXT, email TEXT, created_at TEXT);
                CREATE TABLE posts (id INTEGER, user_id INTEGER, title TEXT, content TEXT, created_at TEXT);
            `)
            if err != nil {
                t.Fatal(err)
            }
        },
        func(tx *sql.Tx) {
            // フィクスチャは自動挿入済み
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

## 📚 API リファレンス

### TestFixture（推奨）

```go
// テスト用の新しいFixtureインスタンスを作成
func NewTestFixture(t *testing.T, db *sql.DB) *TestFixture

// テストセットアップ（YAMLファイルを読み込み）
func (tf *TestFixture) SetupTest(yamlPaths ...string)

// トランザクション内でテスト実行（フィクスチャ自動挿入）
func (tf *TestFixture) RunTest(testFn func(tx *sql.Tx))

// セットアップ後、フィクスチャを挿入してテスト実行
func (tf *TestFixture) RunTestWithSetup(setupFn func(tx *sql.Tx), testFn func(tx *sql.Tx))

// 手動でフィクスチャ挿入タイミングを制御
func (tf *TestFixture) RunTestWithCustomSetup(testFn func(tx *sql.Tx))

// フィクスチャデータを手動挿入（通常は不要）
func (tf *TestFixture) InsertTestData()

// トランザクションが開始されているかを確認
func (tf *TestFixture) HasTransaction() bool

// トランザクションインスタンスを取得（高度な用途）
func (tf *TestFixture) GetTransaction() *sql.Tx

// テストクリーンアップ
func (tf *TestFixture) TearDownTest()
```

**廃止予定のメソッド（互換性のため残存）**
```go
// 非推奨：RunTestWithSetupまたはRunTestを使用してください
func (tf *TestFixture) ExecInTransaction(query string, args ...interface{})
func (tf *TestFixture) QueryInTransaction(query string, args ...interface{}) *sql.Rows
func (tf *TestFixture) QueryRowInTransaction(query string, args ...interface{}) *sql.Row
```

## 🎯 使い方のベストプラクティス

### 新しいAPI（推奨）

```go
// 1. シンプルなケース（テーブル既存）
fixture.RunTest(func(tx *sql.Tx) {
    // フィクスチャ自動挿入済み
    // テストコードのみ記述
})

// 2. セットアップが必要なケース
fixture.RunTestWithSetup(
    func(tx *sql.Tx) {
        // テーブル作成・セットアップ
    },
    func(tx *sql.Tx) {
        // フィクスチャ自動挿入済み
        // テストコード
    },
)

// 3. 複雑な制御が必要なケース
fixture.RunTestWithCustomSetup(func(tx *sql.Tx) {
    // テーブル作成
    // 手動でフィクスチャ挿入
    fixture.InsertTestData()
    // テストコード
})
```

### 🆚 新旧API比較

| 項目                 | 旧API                           | 新API              |
| -------------------- | ------------------------------- | ------------------ |
| フィクスチャ挿入     | `fixture.InsertTestData()` 必須 | 自動実行           |
| トランザクション取得 | `fixture.GetTransaction()`      | 引数で直接受け取り |
| SQL実行              | `fixture.ExecInTransaction()`   | `tx.Exec()`        |
| エラーハンドリング   | ヘルパーメソッド内で自動        | 明示的制御         |
| 可読性               | 冗長                            | 簡潔               |
| 柔軟性               | 限定的                          | 高い               |

### 💡 移行ガイド

```go
// 旧API
fixture.RunTest(func() {
    fixture.ExecInTransaction("CREATE TABLE ...")
    fixture.InsertTestData()
    rows := fixture.QueryInTransaction("SELECT ...")
})

// 新API
fixture.RunTestWithSetup(
    func(tx *sql.Tx) {
        tx.Exec("CREATE TABLE ...")
    },
    func(tx *sql.Tx) {
        rows, _ := tx.Query("SELECT ...")
    },
)
```

### Fixture（低レベルAPI）

```go
// 新しいFixtureインスタンスを作成
func New(config Config) *Fixture

// YAMLファイルから読み込み
func (f *Fixture) LoadFromFile(filepath string) error

// YAMLデータから読み込み
func (f *Fixture) LoadFromYAML(data []byte) error

// フィクスチャ挿入
func (f *Fixture) InsertFixtures() error

// トランザクション管理
func (f *Fixture) BeginTransaction() error
func (f *Fixture) CommitTransaction() error
func (f *Fixture) RollbackTransaction() error

// トランザクション内で関数実行
func (f *Fixture) WithTransaction(fn func() error) error
```

## ⚙️ 設定

### 推奨API（TestFixture）

推奨API `NewTestFixture()` を使用する場合、設定は自動的に最適化されます：

```go
// 自動設定：AutoRollback = true（テスト用途に最適）
fixture := yamlfix.NewTestFixture(t, db)
```

### 低レベルAPI（Fixture）

低レベルAPI `New()` を使用する場合は、手動で `Config` を設定できます：

```go
// 手動設定例1: テスト用途（自動ロールバック有効）
config := yamlfix.Config{
    DB:           db,
    AutoRollback: true, // テスト後に自動でロールバック
}
fixture := yamlfix.New(config)

// 手動設定例2: 本番用途（手動コミット）
config := yamlfix.Config{
    DB:           db,
    AutoRollback: false, // 手動でコミット/ロールバックを制御
}
fixture := yamlfix.New(config)

// 低レベルAPIでの使用例
err := fixture.WithTransaction(func() error {
    return fixture.InsertFixtures()
}) // AutoRollback=falseの場合は自動コミット
```

### Config フィールド

```go
type Config struct {
    DB           *sql.DB // データベース接続
    AutoRollback bool    // 自動ロールバック有効化
}
```

| フィールド     | 説明                                                                     | 推奨設定                        |
| -------------- | ------------------------------------------------------------------------ | ------------------------------- |
| `DB`           | データベース接続                                                         | 必須                            |
| `AutoRollback` | `true`: テスト後自動ロールバック<br>`false`: 手動でコミット/ロールバック | テスト: `true`<br>本番: `false` |

**💡 ヒント**: ほとんどの場合、`NewTestFixture()` の自動設定で十分です。

## 🗄️ サポートするデータベース

- **SQLite** （テスト環境におすすめ）
- **MySQL**
- **PostgreSQL**
- その他 `database/sql` 対応データベース

## 📁 プロジェクト構成例

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

## 🤝 貢献

プルリクエストやIssueは歓迎します！

1. このリポジトリをフォーク
2. フィーチャーブランチを作成 (`git checkout -b feature/amazing-feature`)
3. 変更をコミット (`git commit -m 'Add some amazing feature'`)
4. ブランチにプッシュ (`git push origin feature/amazing-feature`)
5. プルリクエストを作成

## 📄 ライセンス

MIT License

## 🔗 関連リンク

- [Go言語公式サイト](https://golang.org/)
- [database/sql パッケージ](https://pkg.go.dev/database/sql)
- [YAML仕様](https://yaml.org/) 
