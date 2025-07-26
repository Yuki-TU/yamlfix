package example

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Yuki-TU/yamlfix"
	_ "github.com/mattn/go-sqlite3"
)

func TestRepository(t *testing.T) {
	// 共通のセットアップ
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

		t.Run("CreateUser", func(t *testing.T) {
			tests := []struct {
				name     string
				user     User
				wantErr  bool
				validate func(t *testing.T, created User, original User)
			}{
				{
					name: "正常なユーザー作成",
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
				{
					name: "日本語名のユーザー作成",
					user: User{Name: "田中花子", Email: "tanaka@example.com"},
					validate: func(t *testing.T, created User, original User) {
						if created.Name != original.Name {
							t.Errorf("日本語名 - 期待値: %s, 実際の値: %s", original.Name, created.Name)
						}
					},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					created, err := repo.CreateUser(ctx, tx, tt.user)
					if (err != nil) != tt.wantErr {
						t.Fatalf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
					}
					if !tt.wantErr && tt.validate != nil {
						tt.validate(t, created, tt.user)
					}
				})
			}
		})

		t.Run("GetUser", func(t *testing.T) {
			// テストデータを事前に挿入
			testTime := time.Now()
			fixture.ExecInTransaction(`
				INSERT INTO users (id, name, email, created_at) 
				VALUES (100, 'テストユーザー', 'test@example.com', ?)
			`, testTime)

			tests := []struct {
				name     string
				userID   uint64
				want     User
				wantErr  bool
				validate func(t *testing.T, got User, want User)
			}{
				{
					name:   "存在するユーザーの取得",
					userID: 100,
					want:   User{ID: 100, Name: "テストユーザー", Email: "test@example.com"},
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
						if got.CreatedAt.Unix() != testTime.Unix() {
							t.Errorf("作成時刻 - 期待値: %v, 実際の値: %v", testTime, got.CreatedAt)
						}
					},
				},
				{
					name:   "存在しないユーザーの取得",
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

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := repo.GetUser(ctx, tx, tt.userID)
					if (err != nil) != tt.wantErr {
						t.Fatalf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
					}
					if !tt.wantErr && tt.validate != nil {
						tt.validate(t, got, tt.want)
					}
				})
			}
		})

		t.Run("CreateAndGetUser統合テスト", func(t *testing.T) {
			tests := []struct {
				name string
				user User
			}{
				{
					name: "作成して取得の統合テスト",
					user: User{Name: "統合テストユーザー", Email: "integration@example.com"},
				},
				{
					name: "日本語ユーザーの統合テスト",
					user: User{Name: "鈴木一郎", Email: "suzuki@example.com"},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
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
				})
			}
		})
	})
}
