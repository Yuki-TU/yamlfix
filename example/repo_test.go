package example

import (
	"database/sql"
	"testing"

	"github.com/Yuki-TU/yamlfix"
	_ "github.com/mattn/go-sqlite3"
)

// テーブル作成の共通関数
func createUsersTable(tx *sql.Tx, t *testing.T) {
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
}

func TestCreateUser(t *testing.T) {
	// 共通のセットアップ
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	fixture := yamlfix.NewTestFixture(t, db)
	fixture.SetupTest("testdata/users.yaml")

	repo := NewRepository()
	ctx := t.Context()

	tests := map[string]struct {
		user     User
		wantErr  bool
		validate func(t *testing.T, created User, original User)
	}{
		"正常なユーザー作成": {
			user: User{Name: "山田太郎", Email: "yamada@example.com"},
			validate: func(t *testing.T, created User, original User) {
				if created.ID == 0 {
					t.Error("IDが設定されていません")
				}
				if created.Name != original.Name {
					t.Errorf("名前 - 期待値: %s, 実際の値: %s", original.Name, created.Name)
				}
				if created.Email != original.Email {
					t.Errorf("メール - 期待値: %s, 実際の値: %s", original.Email, created.Email)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fixture.RunTestWithSetup(
				func(tx *sql.Tx) {
					createUsersTable(tx, t)
				},
				func(tx *sql.Tx) {
					// フィクスチャデータ（id: 1, 2）は自動挿入済み
					created, err := repo.CreateUser(ctx, tx, tt.user)
					if (err != nil) != tt.wantErr {
						t.Fatalf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
					}
					if !tt.wantErr && tt.validate != nil {
						tt.validate(t, created, tt.user)
					}
				},
			)
		})
	}
}

func TestGetUser(t *testing.T) {
	// 共通のセットアップ
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	fixture := yamlfix.NewTestFixture(t, db)
	fixture.SetupTest("testdata/users.yaml")

	repo := NewRepository()
	ctx := t.Context()

	tests := map[string]struct {
		userID   uint64
		want     User
		wantErr  bool
		validate func(t *testing.T, got User, want User)
	}{
		"存在するユーザーの取得": {
			userID: 1, // フィクスチャデータのid: 1（山田太郎）を使用
			want:   User{ID: 1, Name: "山田太郎", Email: "yamada@example.com"},
			validate: func(t *testing.T, got User, want User) {
				if got.ID != want.ID {
					t.Errorf("ID - 期待値: %d, 実際の値: %d", want.ID, got.ID)
				}
				if got.Name != want.Name {
					t.Errorf("名前 - 期待値: %s, 実際の値: %s", want.Name, got.Name)
				}
				if got.Email != want.Email {
					t.Errorf("メール - 期待値: %s, 実際の値: %s", want.Email, got.Email)
				}
				if got.CreatedAt.IsZero() {
					t.Error("作成時刻が設定されていません")
				}
			},
		},
		"存在しないユーザーの取得": {
			userID: 999,
			want:   User{}, // 空のユーザー
			validate: func(t *testing.T, got User, want User) {
				if got.ID != 0 {
					t.Errorf("存在しないユーザーのID - 期待値: 0, 実際の値: %d", got.ID)
				}
				if got.Name != "" {
					t.Errorf("存在しないユーザーの名前 - 期待値: 空文字, 実際の値: %s", got.Name)
				}
				if got.Email != "" {
					t.Errorf("存在しないユーザーのメール - 期待値: 空文字, 実際の値: %s", got.Email)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fixture.RunTestWithSetup(
				func(tx *sql.Tx) {
					createUsersTable(tx, t)
				},
				func(tx *sql.Tx) {
					// フィクスチャデータが自動挿入済み（id: 1=山田太郎, id: 2=田中花子）
					got, err := repo.GetUser(ctx, tx, tt.userID)
					if (err != nil) != tt.wantErr {
						t.Fatalf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
					}
					if !tt.wantErr && tt.validate != nil {
						tt.validate(t, got, tt.want)
					}
				},
			)
		})
	}
}

func TestCreateAndGetUser(t *testing.T) {
	// 共通のセットアップ
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	fixture := yamlfix.NewTestFixture(t, db)
	fixture.SetupTest("testdata/users.yaml")

	repo := NewRepository()
	ctx := t.Context()

	tests := map[string]struct {
		user User
	}{
		"作成して取得の統合テスト": {
			user: User{Name: "統合テストユーザー", Email: "integration@example.com"},
		},
		"日本語ユーザーの統合テスト": {
			user: User{Name: "鈴木一郎", Email: "suzuki@example.com"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fixture.RunTestWithSetup(
				func(tx *sql.Tx) {
					createUsersTable(tx, t)
				},
				func(tx *sql.Tx) {
					// フィクスチャデータ（id: 1, 2）は自動挿入済み

					// ユーザー作成
					created, err := repo.CreateUser(ctx, tx, tt.user)
					if err != nil {
						t.Fatalf("CreateUser() error = %v", err)
					}

					// 作成したユーザーを取得
					retrieved, err := repo.GetUser(ctx, tx, created.ID)
					if err != nil {
						t.Fatalf("GetUser() error = %v", err)
					}

					// 一致確認
					if retrieved.ID != created.ID {
						t.Errorf("ID - 期待値: %d, 実際の値: %d", created.ID, retrieved.ID)
					}
					if retrieved.Name != created.Name {
						t.Errorf("名前 - 期待値: %s, 実際の値: %s", created.Name, retrieved.Name)
					}
					if retrieved.Email != created.Email {
						t.Errorf("メール - 期待値: %s, 実際の値: %s", created.Email, retrieved.Email)
					}
				},
			)
		})
	}
}
