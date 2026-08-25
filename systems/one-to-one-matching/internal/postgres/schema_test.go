//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSchemaAcceptsValidMatchingData(t *testing.T) {
	ctx, conn := connectTestPostgres(t)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer tx.Rollback(context.Background())

	statements := []struct {
		name  string
		query string
	}{
		{
			name:  "既知ユーザーを保存する",
			query: `INSERT INTO users (id) VALUES ('alice'), ('bob')`,
		},
		{
			name:  "両方向のLikeを保存する",
			query: `INSERT INTO likes (sender_id, receiver_id) VALUES ('alice', 'bob'), ('bob', 'alice')`,
		},
		{
			name:  "正規順のMatchを保存する",
			query: `INSERT INTO matches (user_low_id, user_high_id) VALUES ('alice', 'bob')`,
		},
	}

	for _, stmt := range statements {
		if _, err := tx.Exec(ctx, stmt.query); err != nil {
			t.Fatalf("%s: Exec() error = %v", stmt.name, err)
		}
	}

	counts := []struct {
		name  string
		query string
		want  int
	}{
		{
			name:  "両方向Like",
			query: `SELECT count(*) FROM likes WHERE (sender_id, receiver_id) IN (('alice', 'bob'), ('bob', 'alice'))`,
			want:  2,
		},
		{
			name:  "正規順Match",
			query: `SELECT count(*) FROM matches WHERE user_low_id = 'alice' AND user_high_id = 'bob'`,
			want:  1,
		},
	}

	for _, count := range counts {
		var got int
		if err := tx.QueryRow(ctx, count.query).Scan(&got); err != nil {
			t.Fatalf("%s: QueryRow() error = %v", count.name, err)
		}
		if got != count.want {
			t.Fatalf("%s count = %d, want %d", count.name, got, count.want)
		}
	}
}

func connectTestPostgres(t *testing.T) (context.Context, *pgx.Conn) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgx.Connect() error = %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(context.Background()); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	return ctx, conn
}

func TestSchemaRejectsEmptyUserAndUnknownReferences(t *testing.T) {
	const (
		checkViolation      = "23514" // check constraint violation
		foreignKeyViolation = "23503" // foreign key violation
	)

	tests := []struct {
		name           string
		setupQuery     string
		query          string
		wantCode       string
		wantConstraint string
	}{
		{
			name:           "空のUserID",
			query:          `INSERT INTO users (id) VALUES ('')`,
			wantCode:       checkViolation,
			wantConstraint: "users_id_not_empty",
		},
		{
			name:           "Likeのsenderが存在しない",
			setupQuery:     `INSERT INTO users (id) VALUES ('bob')`,
			query:          `INSERT INTO likes (sender_id, receiver_id) VALUES ('alice', 'bob')`,
			wantCode:       foreignKeyViolation,
			wantConstraint: "likes_sender_fk",
		},
		{
			name:           "Likeのreceiverが存在しない",
			setupQuery:     `INSERT INTO users (id) VALUES ('alice')`,
			query:          `INSERT INTO likes (sender_id, receiver_id) VALUES ('alice', 'bob')`,
			wantCode:       foreignKeyViolation,
			wantConstraint: "likes_receiver_fk",
		},
		{
			name:           "Matchのlow側が存在しない",
			setupQuery:     `INSERT INTO users (id) VALUES ('bob')`,
			query:          `INSERT INTO matches (user_low_id, user_high_id) VALUES ('alice', 'bob')`,
			wantCode:       foreignKeyViolation,
			wantConstraint: "matches_user_low_fk",
		},
		{
			name:           "Matchのhigh側が存在しない",
			setupQuery:     `INSERT INTO users (id) VALUES ('alice')`,
			query:          `INSERT INTO matches (user_low_id, user_high_id) VALUES ('alice', 'bob')`,
			wantCode:       foreignKeyViolation,
			wantConstraint: "matches_user_high_fk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, conn := connectTestPostgres(t)

			tx, err := conn.Begin(ctx)
			if err != nil {
				t.Fatalf("Begin() error = %v", err)
			}
			defer tx.Rollback(context.Background())

			if tt.setupQuery != "" {
				if _, err := tx.Exec(ctx, tt.setupQuery); err != nil {
					t.Fatalf("setup Exec() error = %v", err)
				}
			}

			_, err = tx.Exec(ctx, tt.query)
			if err == nil {
				t.Fatal("Exec() error = nil, want constraint violation")
			}

			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				t.Fatalf("Exec() error = %T, want *pgconn.PgError", err)
			}

			if pgErr.Code != tt.wantCode {
				t.Fatalf("SQLSTATE = %q, want %q", pgErr.Code, tt.wantCode)
			}
			if pgErr.ConstraintName != tt.wantConstraint {
				t.Fatalf(
					"constraint = %q, want %q",
					pgErr.ConstraintName,
					tt.wantConstraint,
				)
			}
		})
	}
}

func TestLikesConstraints(t *testing.T) {
	const (
		checkViolation  = "23514"
		uniqueViolation = "23505"
	)

	tests := []struct {
		name           string
		seedSender     string
		seedReceiver   string
		sender         string
		receiver       string
		wantCode       string
		wantConstraint string
		wantCount      int
	}{
		{
			name:           "自己Likeを拒否する",
			sender:         "alice",
			receiver:       "alice",
			wantCode:       checkViolation,
			wantConstraint: "likes_not_self",
		},
		{
			name:           "同方向の重複Likeを拒否する",
			seedSender:     "alice",
			seedReceiver:   "bob",
			sender:         "alice",
			receiver:       "bob",
			wantCode:       uniqueViolation,
			wantConstraint: "likes_pkey",
		},
		{
			name:         "逆方向のLikeを許可する",
			seedSender:   "alice",
			seedReceiver: "bob",
			sender:       "bob",
			receiver:     "alice",
			wantCount:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, conn := connectTestPostgres(t)

			tx, err := conn.Begin(ctx)
			if err != nil {
				t.Fatalf("Begin() error = %v", err)
			}
			defer tx.Rollback(context.Background())

			if _, err := tx.Exec(
				ctx,
				`INSERT INTO users (id) VALUES ('alice'), ('bob')`,
			); err != nil {
				t.Fatalf("insert users: Exec() error = %v", err)
			}

			if tt.seedSender != "" {
				if _, err := tx.Exec(
					ctx,
					`INSERT INTO likes (sender_id, receiver_id) VALUES ($1, $2)`,
					tt.seedSender,
					tt.seedReceiver,
				); err != nil {
					t.Fatalf("seed like: Exec() error = %v", err)
				}
			}

			_, err = tx.Exec(
				ctx,
				`INSERT INTO likes (sender_id, receiver_id) VALUES ($1, $2)`,
				tt.sender,
				tt.receiver,
			)

			if tt.wantCode != "" {
				if err == nil {
					t.Fatal("Exec() error = nil, want constraint violation")
				}

				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) {
					t.Fatalf("Exec() error = %T, want *pgconn.PgError", err)
				}
				if pgErr.Code != tt.wantCode {
					t.Fatalf("SQLSTATE = %q, want %q", pgErr.Code, tt.wantCode)
				}
				if pgErr.ConstraintName != tt.wantConstraint {
					t.Fatalf(
						"constraint = %q, want %q",
						pgErr.ConstraintName,
						tt.wantConstraint,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("insert reverse like: Exec() error = %v", err)
			}

			var got int
			err = tx.QueryRow(
				ctx,
				`SELECT count(*)
				 FROM likes
				 WHERE (sender_id, receiver_id)
				 IN (('alice', 'bob'), ('bob', 'alice'))`,
			).Scan(&got)
			if err != nil {
				t.Fatalf("count likes: QueryRow() error = %v", err)
			}
			if got != tt.wantCount {
				t.Fatalf("like count = %d, want %d", got, tt.wantCount)
			}
		})
	}
}

func TestMatchesConstraints(t *testing.T) {
	const (
		checkViolation  = "23514"
		uniqueViolation = "23505"
	)

	tests := []struct {
		name           string
		seedLow        string
		seedHigh       string
		low            string
		high           string
		wantCode       string
		wantConstraint string
	}{
		{
			name:           "自己Matchを拒否する",
			low:            "alice",
			high:           "alice",
			wantCode:       checkViolation,
			wantConstraint: "matches_users_ordered",
		},
		{
			name:           "逆順Matchを拒否する",
			low:            "bob",
			high:           "alice",
			wantCode:       checkViolation,
			wantConstraint: "matches_users_ordered",
		},
		{
			name:           "同じ正規順Pairの重複を拒否する",
			seedLow:        "alice",
			seedHigh:       "bob",
			low:            "alice",
			high:           "bob",
			wantCode:       uniqueViolation,
			wantConstraint: "matches_pkey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, conn := connectTestPostgres(t)

			tx, err := conn.Begin(ctx)
			if err != nil {
				t.Fatalf("Begin() error = %v", err)
			}
			defer tx.Rollback(context.Background())

			if _, err := tx.Exec(
				ctx,
				`INSERT INTO users (id) VALUES ('alice'), ('bob')`,
			); err != nil {
				t.Fatalf("insert users: Exec() error = %v", err)
			}

			if tt.seedLow != "" {
				if _, err := tx.Exec(
					ctx,
					`INSERT INTO matches (user_low_id, user_high_id) VALUES ($1, $2)`,
					tt.seedLow,
					tt.seedHigh,
				); err != nil {
					t.Fatalf("seed match: Exec() error = %v", err)
				}
			}

			_, err = tx.Exec(
				ctx,
				`INSERT INTO matches (user_low_id, user_high_id) VALUES ($1, $2)`,
				tt.low,
				tt.high,
			)
			if err == nil {
				t.Fatal("Exec() error = nil, want constraint violation")
			}

			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) {
				t.Fatalf("Exec() error = %T, want *pgconn.PgError", err)
			}
			if pgErr.Code != tt.wantCode {
				t.Fatalf("SQLSTATE = %q, want %q", pgErr.Code, tt.wantCode)
			}
			if pgErr.ConstraintName != tt.wantConstraint {
				t.Fatalf(
					"constraint = %q, want %q",
					pgErr.ConstraintName,
					tt.wantConstraint,
				)
			}
		})
	}
}
