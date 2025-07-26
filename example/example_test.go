package example

import (
	"database/sql"
	"testing"

	"github.com/Yuki-TU/yamlfix"
	_ "github.com/mattn/go-sqlite3"
)

// TestSingleTableYAML はテーブル名.yamlの形式をテストする
func TestSingleTableYAML(t *testing.T) {
	// SQLiteのメモリデータベースを使用
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// テストフィクスチャの初期化
	fixture := yamlfix.NewTestFixture(t, db)
	fixture.SetupTest("testdata/users.yaml")
	defer fixture.TearDownTest()

	// トランザクション内でテスト実行
	fixture.RunTest(func() {
		// テーブル作成（トランザクション内で実行）
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

		// ユーザーデータが正しく挿入されているかテスト
		var count int
		err = fixture.QueryRowInTransaction("SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}

		if count != 2 {
			t.Errorf("期待値: 2, 実際の値: %d", count)
		}

		// 特定のユーザーをテスト
		var name, email string
		err = fixture.QueryRowInTransaction("SELECT name, email FROM users WHERE id = 1").Scan(&name, &email)
		if err != nil {
			t.Fatal(err)
		}

		if name != "山田太郎" {
			t.Errorf("期待値: 山田太郎, 実際の値: %s", name)
		}
		if email != "yamada@example.com" {
			t.Errorf("期待値: yamada@example.com, 実際の値: %s", email)
		}
	})
}

// TestMultipleTableFiles は複数のテーブル名.yamlファイルをテストする
func TestMultipleTableFiles(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// テストフィクスチャの初期化
	fixture := yamlfix.NewTestFixture(t, db)
	fixture.SetupTest("testdata/users.yaml", "testdata/posts.yaml")
	defer fixture.TearDownTest()

	fixture.RunTest(func() {
		// テーブル作成（トランザクション内で実行）
		fixture.ExecInTransaction(`
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

		// フィクスチャデータを挿入
		fixture.InsertTestData()

		// usersテーブルのテスト
		var userCount int
		err = fixture.QueryRowInTransaction("SELECT COUNT(*) FROM users").Scan(&userCount)
		if err != nil {
			t.Fatal(err)
		}
		if userCount != 2 {
			t.Errorf("users テーブル - 期待値: 2, 実際の値: %d", userCount)
		}

		// postsテーブルのテスト
		var postCount int
		err = fixture.QueryRowInTransaction("SELECT COUNT(*) FROM posts").Scan(&postCount)
		if err != nil {
			t.Fatal(err)
		}
		if postCount != 2 {
			t.Errorf("posts テーブル - 期待値: 2, 実際の値: %d", postCount)
		}

		// JOIN クエリのテスト
		var title, userName string
		err = fixture.QueryRowInTransaction(`
			SELECT p.title, u.name 
			FROM posts p 
			JOIN users u ON p.user_id = u.id 
			WHERE p.id = 1
		`).Scan(&title, &userName)
		if err != nil {
			t.Fatal(err)
		}

		if title != "最初の投稿" {
			t.Errorf("期待値: 最初の投稿, 実際の値: %s", title)
		}
		if userName != "山田太郎" {
			t.Errorf("期待値: 山田太郎, 実際の値: %s", userName)
		}
	})
}
