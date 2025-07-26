package yamlfix

import (
	"database/sql"
	"testing"
)

// TestFixture はテスト用のフィクスチャヘルパー
type TestFixture struct {
	*Fixture
	t *testing.T
}

// NewTestFixture はテスト用の新しいFixtureインスタンスを作成する
func NewTestFixture(t *testing.T, db *sql.DB) *TestFixture {
	return &TestFixture{
		Fixture: New(Config{
			DB:           db,
			AutoRollback: true, // テスト時は常に自動ロールバック
		}),
		t: t,
	}
}

// SetupTest はテストのセットアップを行う
func (tf *TestFixture) SetupTest(yamlPaths ...string) {
	tf.t.Helper()

	// ファイルまたはディレクトリから読み込み
	for _, path := range yamlPaths {
		if err := tf.LoadFromFile(path); err != nil {
			tf.t.Fatalf("フィクスチャの読み込みに失敗しました: %v", err)
		}
	}
}

// RunTest はトランザクション内でテストを実行する
func (tf *TestFixture) RunTest(testFn func()) {
	tf.t.Helper()

	// トランザクション開始
	err := tf.BeginTransaction()
	if err != nil {
		tf.t.Fatalf("トランザクション開始エラー: %v", err)
	}

	defer func() {
		if tf.autoRollback {
			tf.RollbackTransaction()
		}
	}()

	// テストコードを実行（この中でInsertFixturesを呼ぶ）
	testFn()
}

// InsertTestData はテスト内でフィクスチャデータを挿入する
func (tf *TestFixture) InsertTestData() {
	tf.t.Helper()

	err := tf.InsertFixtures()
	if err != nil {
		tf.t.Fatalf("フィクスチャ挿入エラー: %v", err)
	}
}

// ExecInTransaction はトランザクション内でSQLを実行する
func (tf *TestFixture) ExecInTransaction(query string, args ...interface{}) {
	tf.t.Helper()

	executor := tf.getExecutor()
	_, err := executor.Exec(query, args...)
	if err != nil {
		tf.t.Fatalf("SQL実行エラー: %v", err)
	}
}

// QueryInTransaction はトランザクション内でクエリを実行する
func (tf *TestFixture) QueryInTransaction(query string, args ...interface{}) *sql.Rows {
	tf.t.Helper()

	executor := tf.getExecutor()
	rows, err := executor.Query(query, args...)
	if err != nil {
		tf.t.Fatalf("クエリ実行エラー: %v", err)
	}
	return rows
}

// QueryRowInTransaction はトランザクション内で単一行クエリを実行する
func (tf *TestFixture) QueryRowInTransaction(query string, args ...interface{}) *sql.Row {
	tf.t.Helper()

	executor := tf.getExecutor()
	return executor.QueryRow(query, args...)
}

// HasTransaction はトランザクションが開始されているかを確認する
func (tf *TestFixture) HasTransaction() bool {
	return tf.tx != nil
}

// GetTransaction はトランザクションインスタンスを取得する
func (tf *TestFixture) GetTransaction() *sql.Tx {
	return tf.tx
}

// TearDownTest はテストのクリーンアップを行う
func (tf *TestFixture) TearDownTest() {
	tf.t.Helper()

	if err := tf.CleanUp(); err != nil {
		tf.t.Errorf("テストクリーンアップに失敗しました: %v", err)
	}
}
