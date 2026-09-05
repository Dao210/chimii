package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The product-owned deletion contract. Shared part catalogs are deliberately
// outside workspace ownership; brick_inventory_item is reached via inventory.
var buildCleanupTables = []string{"build_creation", "build_job", "build_session", "brick_inventory", "child_profile", "child_session", "circuit_creation", "circuit_inventory", "circuit_trial"}

func deletionWorkspace(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	id := uuid.NewString()
	if _, err := testPool.Exec(ctx, `INSERT INTO workspace(id,name,slug) VALUES($1,'Cleanup test',$2)`, id, "cleanup-"+id); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, `INSERT INTO member(workspace_id,user_id,role) VALUES($1,$2,'owner')`, id, testUserID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testHandler.Queries.DeleteWorkspace(context.Background(), parseUUID(id)); err != nil {
			t.Error(err)
		}
	})
	return id
}

func seedDeletionProduct(t *testing.T, ws string) {
	t.Helper()
	ctx := context.Background()
	statements := []string{
		`INSERT INTO build_session(workspace_id,creator_user_id,client_request_id,prompt) VALUES($1,$2,gen_random_uuid(),'test')`,
		`INSERT INTO build_job(workspace_id,session_id) SELECT workspace_id,id FROM build_session WHERE workspace_id=$1 AND creator_user_id=$2`,
		`INSERT INTO build_creation(workspace_id,creator_user_id,session_id,title,prompt,archetype,recipe,build_plan,validation,ldraw_mpd,current_step) VALUES($1,$2,gen_random_uuid(),'test','test','robot','{}','{}','{"step_count":2,"part_count":2}','0 test',1)`,
		`INSERT INTO brick_inventory(workspace_id,updated_by,catalog_version) VALUES($1,$2,'test')`,
		`INSERT INTO brick_inventory_item(inventory_id,part_key,color_code,quantity) SELECT id,'3001.dat',4,2 FROM brick_inventory WHERE workspace_id=$1 AND updated_by=$2`,
		`INSERT INTO child_profile(workspace_id,parent_user_id,display_name,pin_hash) VALUES($1,$2,'test','test')`,
		`INSERT INTO child_session(workspace_id,parent_user_id,profile_id,token_hash,expires_at) SELECT workspace_id,parent_user_id,id,id::text,now()+interval '1 hour' FROM child_profile WHERE workspace_id=$1 AND parent_user_id=$2`,
		`INSERT INTO circuit_creation(workspace_id,creator_user_id,actor_key,client_request_id,request_hash,document) VALUES($1,$2,'parent',gen_random_uuid(),'test','{}')`,
		`INSERT INTO circuit_inventory(workspace_id,parent_user_id,kit_id,catalog_version,quantities) VALUES($1,$2,'test','test','{}')`,
		`INSERT INTO circuit_trial(id,workspace_id,parent_user_id,actor_key,creation_id,document_hash,request_hash,hardware_label,result,notes) VALUES(gen_random_uuid(),$1,$2,'parent',gen_random_uuid(),'test','test','test','worked','')`,
	}
	for _, sql := range statements {
		if _, err := testPool.Exec(ctx, sql, ws, testUserID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWorkspaceBuildCleanupManifest(t *testing.T) {
	rows, err := testPool.Query(context.Background(), `SELECT table_name FROM information_schema.columns WHERE table_schema=current_schema() AND column_name='workspace_id' AND table_name ~ '^(build_|brick_|child_|circuit_)'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Fatal(err)
		}
		got = append(got, table)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	want := append([]string(nil), buildCleanupTables...)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("new workspace table needs a cleanup decision: got %v want %v", got, want)
	}
}

func TestWorkspaceBuildCleanupRollbackAndIsolation(t *testing.T) {
	ws, other := deletionWorkspace(t), deletionWorkspace(t)
	seedDeletionProduct(t, ws)
	seedDeletionProduct(t, other)
	ctx := context.Background()
	var inventory string
	if err := testPool.QueryRow(ctx, `SELECT id FROM brick_inventory WHERE workspace_id=$1`, ws).Scan(&inventory); err != nil {
		t.Fatal(err)
	}
	var catalogBefore int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM kit_profile`).Scan(&catalogBefore); err != nil {
		t.Fatal(err)
	}
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	q := db.New(tx)
	if _, err = q.LockWorkspaceForDelete(ctx, parseUUID(ws)); err != nil {
		t.Fatal(err)
	}
	if err = q.DeleteWorkspace(ctx, parseUUID(ws)); err != nil {
		t.Fatal(err)
	}
	// An injected SQL error aborts the same transaction; every deletion must roll back.
	if _, err = tx.Exec(ctx, `SELECT 1/0`); err == nil {
		t.Fatal("expected injected failure")
	}
	_ = tx.Rollback(ctx)
	for _, table := range buildCleanupTables {
		var count int
		if err := testPool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s WHERE workspace_id=$1", table), ws).Scan(&count); err != nil || count != 1 {
			t.Fatalf("rollback %s=%d: %v", table, count, err)
		}
	}
	w := httptest.NewRecorder()
	testHandler.DeleteWorkspace(w, withURLParam(newRequest("DELETE", "/api/workspaces/"+ws, nil), "id", ws))
	if w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, table := range buildCleanupTables {
		var removed, kept int
		if err := testPool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FILTER(WHERE workspace_id=$1),count(*) FILTER(WHERE workspace_id=$2) FROM %s", table), ws, other).Scan(&removed, &kept); err != nil || removed != 0 || kept != 1 {
			t.Fatalf("scope %s: %d %d %v", table, removed, kept, err)
		}
	}
	var remaining, catalogAfter int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM brick_inventory_item WHERE inventory_id=$1`, inventory).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatal("orphan inventory items", remaining, err)
	}
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM kit_profile`).Scan(&catalogAfter); err != nil || catalogAfter != catalogBefore {
		t.Fatal("shared catalog changed", err)
	}
}

func waitDatabaseBlocked(t *testing.T, ctx context.Context, pid uint32) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		if err := testPool.QueryRow(ctx, `SELECT cardinality(pg_blocking_pids($1))>0`, int32(pid)).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("operation did not reach lock barrier")
		case <-ticker.C:
		}
	}
}

func TestWorkspaceBuildDeleteWriteRace(t *testing.T) {
	for _, writerFirst := range []bool{true, false} {
		t.Run(fmt.Sprint(writerFirst), func(t *testing.T) {
			ws := deletionWorkspace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			first, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer first.Rollback(context.Background())
			second, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer second.Rollback(context.Background())
			q1, q2 := db.New(first), db.New(second)
			secondPID := second.Conn().PgConn().PID()
			insert := func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `INSERT INTO circuit_inventory(workspace_id,parent_user_id,kit_id,catalog_version,quantities) VALUES($1,$2,'race','test','{}')`, ws, testUserID)
				return err
			}
			done := make(chan error, 1)
			if writerFirst {
				if err = lockWorkspaceMemberForScopedWrite(ctx, q1, parseUUID(ws), parseUUID(testUserID)); err != nil {
					t.Fatal(err)
				}
				if err = insert(first); err != nil {
					t.Fatal(err)
				}
				go func() {
					_, e := q2.LockWorkspaceForDelete(ctx, parseUUID(ws))
					if e == nil {
						e = q2.DeleteWorkspace(ctx, parseUUID(ws))
					}
					if e == nil {
						e = second.Commit(ctx)
					}
					done <- e
				}()
			} else {
				if _, err = q1.LockWorkspaceForDelete(ctx, parseUUID(ws)); err != nil {
					t.Fatal(err)
				}
				if err = q1.DeleteWorkspace(ctx, parseUUID(ws)); err != nil {
					t.Fatal(err)
				}
				go func() {
					e := lockWorkspaceMemberForScopedWrite(ctx, q2, parseUUID(ws), parseUUID(testUserID))
					if e == nil {
						e = insert(second)
					}
					done <- e
				}()
			}
			waitDatabaseBlocked(t, ctx, secondPID)
			if err = first.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			err = <-done
			if writerFirst && err != nil {
				t.Fatal(err)
			}
			if !writerFirst && !errors.Is(err, pgx.ErrNoRows) {
				t.Fatalf("writer survived deletion: %v", err)
			}
			var count int
			if err := testPool.QueryRow(ctx, `SELECT count(*) FROM circuit_inventory WHERE workspace_id=$1`, ws).Scan(&count); err != nil || count != 0 {
				t.Fatal("orphan write", count, err)
			}
		})
	}
}
