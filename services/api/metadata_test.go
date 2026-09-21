// Metadata separation, proven against the live DB. Plain words: guesses land
// beside the capture (with their source), never inside the user's bytes,
// and die with the item. Runs only with HELMME_TEST_DB set; skips in CI.
package main

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestItemMetadataUpsertLive(t *testing.T) {
	dsn := os.Getenv("HELMME_TEST_DB")
	if dsn == "" {
		t.Skip("no db")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var id string
	if err := pool.QueryRow(ctx, `
	  INSERT INTO items (user_id, source_type, raw_ref, original, title, client_id)
	  VALUES ('local','text','meta probe','{"raw_ref":"meta probe"}','meta probe','phone-test-090') RETURNING id`,
	).Scan(&id); err != nil {
		t.Fatal(err)
	}
	// Same statement shape as fetch.go's guess filing.
	if _, err := pool.Exec(ctx, `
	  INSERT INTO item_metadata (item_id, key, value, source, confidence)
	  VALUES ($1,'fetched_author','Ada','fetch',0.7)
	  ON CONFLICT (item_id, key) DO UPDATE SET value='Ada', updated_at=now()`, id); err != nil {
		t.Fatalf("metadata upsert failed: %v", err)
	}
	var value, source string
	var orig string
	if err := pool.QueryRow(ctx,
		`SELECT value, source FROM item_metadata WHERE item_id=$1 AND key='fetched_author'`,
		id).Scan(&value, &source); err != nil || value != "Ada" || source != "fetch" {
		t.Fatalf("metadata not filed: %v %q %q", err, value, source)
	}
	if err := pool.QueryRow(ctx, `SELECT original::text FROM items WHERE id=$1`, id).Scan(&orig); err != nil {
		t.Fatal(err)
	}
	if orig != `{"raw_ref": "meta probe"}` && orig != `{"raw_ref":"meta probe"}` {
		t.Fatalf("original bytes touched: %s", orig)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM items WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM item_metadata WHERE item_id=$1`, id).Scan(&left); err != nil || left != 0 {
		t.Fatalf("guesses outlived the item: %v count=%d", err, left)
	}
}
