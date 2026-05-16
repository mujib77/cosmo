package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	conn *pgx.Conn
}

func New(databaseURL string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	return &DB{conn: conn}, nil
}

func (db *DB) Close() {
	db.conn.Close(context.Background())
}

type OverviewStats struct {
	DatabaseName    string
	TotalSize       string
	ActiveConns     int
	IdleConns       int
	MaxConns        int
	Uptime          string
	CacheHitRatio   float64
	TransactionsPS  int64
}


type ActiveQuery struct {
	PID      int
	Username string
	Database string
	State    string
	Duration string
	Query    string
}

type WALStats struct {
	CurrentLSN     string
	WALBytesPS     int64
	DeadTuples     int64
	LiveTuples     int64
	AutovacuumCount int64
	CheckpointsPS  int64
}

type LockInfo struct {
	PID        int
	Username   string
	LockType   string
	Granted    bool
	WaitEvent  string
	Query      string
}

func (db *DB) GetOverviewStats(ctx context.Context) (*OverviewStats, error) {
	var stats OverviewStats

	err := db.conn.QueryRow(ctx, `
		SELECT
			current_database(),
			pg_size_pretty(pg_database_size(current_database())),
			count(*) FILTER (WHERE state = 'active'),
			count(*) FILTER (WHERE state = 'idle'),
			current_setting('max_connections')::int,
			date_trunc('second', now() - pg_postmaster_start_time())::text,
			ROUND(
				sum(blks_hit) * 100.0 / NULLIF(sum(blks_hit) + sum(blks_read), 0),
				2
			),
			sum(xact_commit + xact_rollback)
		FROM pg_stat_activity, pg_stat_database
		WHERE pg_stat_database.datname = current_database()
		GROUP BY 1, 2, 6
		LIMIT 1
	`).Scan(
		&stats.DatabaseName,
		&stats.TotalSize,
		&stats.ActiveConns,
		&stats.IdleConns,
		&stats.MaxConns,
		&stats.Uptime,
		&stats.CacheHitRatio,
		&stats.TransactionsPS,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (db *DB) GetActiveQueries(ctx context.Context) ([]ActiveQuery, error) {
	rows, err := db.conn.Query(ctx, `
		SELECT
			pid,
			usename,
			datname,
			state,
			COALESCE(
				date_trunc('second', now() - query_start)::text,
				'0s'
			),
			LEFT(query, 80)
		FROM pg_stat_activity
		WHERE state IS NOT NULL
		AND query NOT LIKE '%pg_stat_activity%'
		ORDER BY query_start DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queries []ActiveQuery
	for rows.Next() {
		var q ActiveQuery
		err := rows.Scan(
			&q.PID,
			&q.Username,
			&q.Database,
			&q.State,
			&q.Duration,
			&q.Query,
		)
		if err != nil {
			continue
		}
		queries = append(queries, q)
	}
	return queries, nil
}

func (db *DB) GetWALStats(ctx context.Context) (*WALStats, error) {
	var stats WALStats

	err := db.conn.QueryRow(ctx, `
		SELECT
			pg_current_wal_lsn()::text,
			pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0'),
			sum(n_dead_tup),
			sum(n_live_tup),
			sum(n_autoanalyze_count),
			checkpoints_timed + checkpoints_req
		FROM pg_stat_user_tables, pg_stat_bgwriter
		GROUP BY 1, 2, 6
		LIMIT 1
	`).Scan(
		&stats.CurrentLSN,
		&stats.WALBytesPS,
		&stats.DeadTuples,
		&stats.LiveTuples,
		&stats.AutovacuumCount,
		&stats.CheckpointsPS,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (db *DB) GetLocks(ctx context.Context) ([]LockInfo, error) {
	rows, err := db.conn.Query(ctx, `
		SELECT
			a.pid,
			a.usename,
			l.locktype,
			l.granted,
			COALESCE(a.wait_event, 'none'),
			LEFT(a.query, 60)
		FROM pg_locks l
		JOIN pg_stat_activity a ON l.pid = a.pid
		WHERE a.query NOT LIKE '%pg_locks%'
		ORDER BY l.granted ASC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locks []LockInfo
	for rows.Next() {
		var l LockInfo
		err := rows.Scan(
			&l.PID,
			&l.Username,
			&l.LockType,
			&l.Granted,
			&l.WaitEvent,
			&l.Query,
		)
		if err != nil {
			continue
		}
		locks = append(locks, l)
	}
	return locks, nil
}
