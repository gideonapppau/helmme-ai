package main

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Throwaway repro: runs the exact upload-index statement against the live
// compose DB. Delete after diagnosis. Run with HELMME_REPRO_DB set.
func TestDBReproIndex(t *testing.T) {
	dsn := os.Getenv("HELMME_REPRO_DB")
	if dsn == "" {
		t.Skip("no db")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	text := "Hello helmme Postgres pooling test"
	_, err = pool.Exec(ctx, `
	  INSERT INTO item_contents (item_id, extracted_text, search_tsv)
	  VALUES ($1,$2,to_tsvector('english',$2 || ' ' || $3))`,
		"00000000-0000-0000-0000-000000000000", text, searchNorm(text))
	t.Logf("exec err=%v", err)
}
