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

	// セットアップとテストを分離した実行
	fixture.RunTestWithSetup(
		func(tx *sql.Tx) {
			// テーブル作成（セットアップ段階）
			_, err := tx.Exec(`
				CREATE TABLE users (
					id INTEGER PRIMARY KEY,
					name TEXT NOT NULL,
					email TEXT NOT NULL,
					created_at TEXT
				)
			`)
			if err != nil {
				t.Fatal(err)
			}
		},
		func(tx *sql.Tx) {
			// テスト段階（フィクスチャは自動挿入済み）
			// ユーザーデータが正しく挿入されているかテスト
			var count int
			err = tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
			if err != nil {
				t.Fatal(err)
			}

			if count != 2 {
				t.Errorf("expected: 2, got: %d", count)
			}

			// 特定のユーザーをテスト
			var name, email string
			err = tx.QueryRow("SELECT name, email FROM users WHERE id = 1").Scan(&name, &email)
			if err != nil {
				t.Fatal(err)
			}

			if name != "山田太郎" {
				t.Errorf("expected: 山田太郎, got: %s", name)
			}
			if email != "yamada@example.com" {
				t.Errorf("expected: yamada@example.com, got: %s", email)
			}
		},
	)
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

	fixture.RunTestWithSetup(
		func(tx *sql.Tx) {
			// テーブル作成（セットアップ段階）
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
			// テスト段階（フィクスチャは自動挿入済み）

			// usersテーブルのテスト
			var userCount int
			err = tx.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
			if err != nil {
				t.Fatal(err)
			}
			if userCount != 2 {
				t.Errorf("users table - expected: 2, got: %d", userCount)
			}

			// postsテーブルのテスト
			var postCount int
			err = tx.QueryRow("SELECT COUNT(*) FROM posts").Scan(&postCount)
			if err != nil {
				t.Fatal(err)
			}
			if postCount != 2 {
				t.Errorf("posts table - expected: 2, got: %d", postCount)
			}

			// JOIN クエリのテスト
			var title, userName string
			err = tx.QueryRow(`
				SELECT p.title, u.name 
				FROM posts p 
				JOIN users u ON p.user_id = u.id 
				WHERE p.id = 1
			`).Scan(&title, &userName)
			if err != nil {
				t.Fatal(err)
			}

			if title != "最初の投稿" {
				t.Errorf("expected: 最初の投稿, got: %s", title)
			}
			if userName != "山田太郎" {
				t.Errorf("expected: 山田太郎, got: %s", userName)
			}
		},
	)
}
