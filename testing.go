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
			tf.t.Fatalf("failed to load fixtures: %v", err)
		}
	}
}

// RunTest はトランザクション内でテストを実行する（フィクスチャ自動挿入）
func (tf *TestFixture) RunTest(testFn func(tx *sql.Tx)) {
	tf.RunTestWithSetup(nil, testFn)
}

// RunTestWithSetup はセットアップ（テーブル作成等）を行った後、フィクスチャを挿入してテストを実行する
func (tf *TestFixture) RunTestWithSetup(setupFn func(tx *sql.Tx), testFn func(tx *sql.Tx)) {
	tf.t.Helper()

	// トランザクション開始
	err := tf.BeginTransaction()
	if err != nil {
		tf.t.Fatalf("failed to start transaction: %v", err)
	}

	defer func() {
		if tf.autoRollback {
			tf.RollbackTransaction()
		}
	}()

	// セットアップ段階（テーブル作成等）
	if setupFn != nil {
		setupFn(tf.tx)
	}

	// フィクスチャデータを自動挿入
	err = tf.InsertFixtures()
	if err != nil {
		tf.t.Fatalf("failed to insert fixtures: %v", err)
	}

	// テストコードを実行
	testFn(tf.tx)
}

// RunTestWithCustomSetup は手動でフィクスチャ挿入タイミングを制御したい場合に使用する
func (tf *TestFixture) RunTestWithCustomSetup(testFn func(tx *sql.Tx)) {
	tf.t.Helper()

	// トランザクション開始
	err := tf.BeginTransaction()
	if err != nil {
		tf.t.Fatalf("failed to start transaction: %v", err)
	}

	defer func() {
		if tf.autoRollback {
			tf.RollbackTransaction()
		}
	}()

	// フィクスチャデータは自動挿入しない（手動制御）
	testFn(tf.tx)
}

// InsertTestData はテスト内でフィクスチャデータを挿入する
func (tf *TestFixture) InsertTestData() {
	tf.t.Helper()

	err := tf.InsertFixtures()
	if err != nil {
		tf.t.Fatalf("failed to insert fixtures: %v", err)
	}
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
		tf.t.Errorf("failed to cleanup test: %v", err)
	}
}
