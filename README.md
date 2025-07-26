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

    fixture.RunTest(func() {
        // テーブル作成
        fixture.ExecInTransaction(`
            CREATE TABLE users (
                id INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                email TEXT NOT NULL,
                created_at TEXT
            )
        `)

        // フィクスチャデータを挿入
        fixture.InsertTestData()

        // テスト実行
        users, err := repo.GetAllUsers(fixture.GetTransaction())
        if err != nil {
            t.Fatal(err)
        }

        if len(users) != 2 {
            t.Errorf("期待値: 2, 実際の値: %d", len(users))
        }
    })
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
    fixture.SetupTest()
    defer fixture.TearDownTest()

    repo := NewRepository()
    ctx := t.Context()

    fixture.RunTest(func() {
        // テーブル作成
        fixture.ExecInTransaction(`
            CREATE TABLE users (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                name TEXT NOT NULL,
                email TEXT NOT NULL,
                created_at DATETIME
            )
        `)

        tx := fixture.GetTransaction()

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
    })
}
```

### 4. 複数テーブル形式（互換性サポート）

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

## 📚 API リファレンス

### TestFixture（推奨）

```go
// テスト用の新しいFixtureインスタンスを作成
func NewTestFixture(t *testing.T, db *sql.DB) *TestFixture

// テストセットアップ（YAMLファイルを読み込み）
func (tf *TestFixture) SetupTest(yamlPaths ...string)

// トランザクション内でテスト実行
func (tf *TestFixture) RunTest(testFn func())

// フィクスチャデータを挿入
func (tf *TestFixture) InsertTestData()

// トランザクション内でSQLを実行
func (tf *TestFixture) ExecInTransaction(query string, args ...interface{})

// トランザクション内でクエリを実行
func (tf *TestFixture) QueryInTransaction(query string, args ...interface{}) *sql.Rows

// トランザクション内で単一行クエリを実行
func (tf *TestFixture) QueryRowInTransaction(query string, args ...interface{}) *sql.Row

// トランザクションインスタンスを取得（リポジトリパターン用）
func (tf *TestFixture) GetTransaction() *sql.Tx

// トランザクションが開始されているかを確認
func (tf *TestFixture) HasTransaction() bool

// テストクリーンアップ
func (tf *TestFixture) TearDownTest()
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

```go
type Config struct {
    DB           *sql.DB // データベース接続
    AutoRollback bool    // 自動ロールバック有効化
}
```

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
