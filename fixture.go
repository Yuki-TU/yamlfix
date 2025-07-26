package yamlfix

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Fixture はYAMLベースのテストフィクスチャを管理する構造体
type Fixture struct {
	db           *sql.DB
	tx           *sql.Tx
	tableOrder   []string
	fixtures     map[string][]map[string]interface{}
	autoRollback bool
}

// Config はFixtureの設定
type Config struct {
	DB           *sql.DB
	AutoRollback bool // テスト後に自動でロールバックするかどうか
}

// New は新しいFixtureインスタンスを作成する
func New(config Config) *Fixture {
	return &Fixture{
		db:           config.DB,
		fixtures:     make(map[string][]map[string]interface{}),
		autoRollback: config.AutoRollback,
	}
}

// LoadFromFile はYAMLファイルからフィクスチャを読み込む
func (f *Fixture) LoadFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("YAMLファイルの読み込みエラー: %w", err)
	}

	return f.LoadFromYAMLWithFilename(data, filepath)
}

// LoadFromYAML はYAMLデータからフィクスチャを読み込む
func (f *Fixture) LoadFromYAML(data []byte) error {
	return f.LoadFromYAMLWithFilename(data, "")
}

// LoadFromYAMLWithFilename はYAMLデータをファイル名情報付きで読み込む
func (f *Fixture) LoadFromYAMLWithFilename(data []byte, filename string) error {
	// まず複数テーブル形式を試行
	var multiTableData map[string][]map[string]interface{}
	if err := yaml.Unmarshal(data, &multiTableData); err == nil {
		// 複数テーブル形式として有効かチェック
		if f.isMultiTableFormat(multiTableData) {
			return f.loadMultiTableData(multiTableData)
		}
	}

	// 単一テーブル形式を試行
	var singleTableData []map[string]interface{}
	if err := yaml.Unmarshal(data, &singleTableData); err != nil {
		return fmt.Errorf("YAMLのパースエラー: %w", err)
	}

	// ファイル名からテーブル名を推測
	tableName := f.extractTableNameFromFilename(filename)
	if tableName == "" {
		return fmt.Errorf("テーブル名を推測できません。ファイル名を指定するか、複数テーブル形式を使用してください")
	}

	return f.loadSingleTableData(tableName, singleTableData)
}

// isMultiTableFormat は複数テーブル形式かどうかを判定する
func (f *Fixture) isMultiTableFormat(data map[string][]map[string]interface{}) bool {
	// 空でない場合は複数テーブル形式とみなす
	return len(data) > 0
}

// extractTableNameFromFilename はファイル名からテーブル名を抽出する
func (f *Fixture) extractTableNameFromFilename(filename string) string {
	if filename == "" {
		return ""
	}

	// ファイル名から拡張子を除いてテーブル名を取得
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// loadMultiTableData は複数テーブル形式のデータを読み込む
func (f *Fixture) loadMultiTableData(yamlData map[string][]map[string]interface{}) error {
	// 既存のデータにマージ
	if f.fixtures == nil {
		f.fixtures = make(map[string][]map[string]interface{})
	}

	for tableName, records := range yamlData {
		f.fixtures[tableName] = records
	}

	// テーブルの順序を更新
	f.updateTableOrder()
	return nil
}

// loadSingleTableData は単一テーブル形式のデータを読み込む
func (f *Fixture) loadSingleTableData(tableName string, records []map[string]interface{}) error {
	// 既存のデータにマージ
	if f.fixtures == nil {
		f.fixtures = make(map[string][]map[string]interface{})
	}

	f.fixtures[tableName] = records

	// テーブルの順序を更新
	f.updateTableOrder()
	return nil
}

// updateTableOrder はテーブルの順序を更新する
func (f *Fixture) updateTableOrder() {
	// 新しいテーブルを順序に追加
	existingTables := make(map[string]bool)
	for _, tableName := range f.tableOrder {
		existingTables[tableName] = true
	}

	for tableName := range f.fixtures {
		if !existingTables[tableName] {
			f.tableOrder = append(f.tableOrder, tableName)
		}
	}
}

// LoadFromDirectory は指定ディレクトリ内の全YAMLファイルを読み込む
func (f *Fixture) LoadFromDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml")) {
			return f.LoadFromFile(path)
		}

		return nil
	})
}

// BeginTransaction はトランザクションを開始する
func (f *Fixture) BeginTransaction() error {
	if f.tx != nil {
		return fmt.Errorf("すでにトランザクションが開始されています")
	}

	tx, err := f.db.Begin()
	if err != nil {
		return fmt.Errorf("トランザクション開始エラー: %w", err)
	}

	f.tx = tx
	return nil
}

// CommitTransaction はトランザクションをコミットする
func (f *Fixture) CommitTransaction() error {
	if f.tx == nil {
		return fmt.Errorf("トランザクションが開始されていません")
	}

	err := f.tx.Commit()
	f.tx = nil
	return err
}

// RollbackTransaction はトランザクションをロールバックする
func (f *Fixture) RollbackTransaction() error {
	if f.tx == nil {
		return fmt.Errorf("トランザクションが開始されていません")
	}

	err := f.tx.Rollback()
	f.tx = nil
	return err
}

// InsertFixtures はフィクスチャデータをデータベースに挿入する
func (f *Fixture) InsertFixtures() error {
	executor := f.getExecutor()

	for _, tableName := range f.tableOrder {
		records := f.fixtures[tableName]
		if len(records) == 0 {
			continue
		}

		if err := f.insertTable(executor, tableName, records); err != nil {
			return fmt.Errorf("テーブル %s への挿入エラー: %w", tableName, err)
		}
	}

	return nil
}

// CleanUp はフィクスチャのクリーンアップを行う
func (f *Fixture) CleanUp() error {
	if f.autoRollback && f.tx != nil {
		return f.RollbackTransaction()
	}
	return nil
}

// WithTransaction はトランザクション内でコールバック関数を実行する
func (f *Fixture) WithTransaction(fn func() error) error {
	if err := f.BeginTransaction(); err != nil {
		return err
	}

	defer func() {
		if f.autoRollback {
			f.RollbackTransaction()
		}
	}()

	if err := f.InsertFixtures(); err != nil {
		f.RollbackTransaction()
		return err
	}

	if err := fn(); err != nil {
		f.RollbackTransaction()
		return err
	}

	if !f.autoRollback {
		return f.CommitTransaction()
	}

	return nil
}

// Executor はSQL実行用のインターフェース
type Executor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// getExecutor は実行用のインターフェースを取得する
func (f *Fixture) getExecutor() Executor {
	if f.tx != nil {
		return f.tx
	}
	return f.db
}

// insertTable は指定テーブルにレコードを挿入する
func (f *Fixture) insertTable(executor Executor, tableName string, records []map[string]interface{}) error {

	if len(records) == 0 {
		return nil
	}

	// カラム名を取得
	columns := make([]string, 0)
	for col := range records[0] {
		columns = append(columns, col)
	}

	// プレースホルダーを作成
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// レコードを順次挿入
	for _, record := range records {
		values := make([]interface{}, len(columns))
		for i, col := range columns {
			values[i] = record[col]
		}

		if _, err := executor.Exec(query, values...); err != nil {
			return fmt.Errorf("レコード挿入エラー: %w", err)
		}
	}

	return nil
}
