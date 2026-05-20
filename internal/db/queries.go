package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	conn *pgxpool.Pool
	prevLSN int64
	prevTime time.Time
}

func New(databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	return &DB{
		conn: pool,
		prevTime: time.Now(),
	}, nil
}

func (db *DB) Close() {
	db.conn.Close()
}

type OverviewStats struct {
	DatabaseName    string
	Version		    string
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
	WALRateMBPS     float64
}

type LockInfo struct {
	PID        int
	Username   string
	LockType   string
	Granted    bool
	WaitEvent  string
	Query      string
	Table      string
}

func (db *DB) GetOverviewStats(ctx context.Context) (*OverviewStats, error) {
	var stats OverviewStats

err := db.conn.QueryRow(ctx, `
    SELECT
        current_database(),
        split_part(version(), ' ', 2),
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
    GROUP BY 1, 2, 3, 6, 7
    LIMIT 1
`).Scan(
    &stats.DatabaseName,
    &stats.Version,
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
        0::bigint,
        0::bigint,
		0::bigint,
        num_timed + num_requested
    FROM pg_stat_checkpointer
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

	now := time.Now()
	elapsed := now.Sub(db.prevTime).Seconds()
	if elapsed > 0 && db.prevLSN > 0 {
		bytesDiff := stats.WALBytesPS - db.prevLSN
		stats.WALRateMBPS = float64(bytesDiff) / elapsed / 1024 / 1024
	}
	db.prevLSN = stats.WALBytesPS
	db.prevTime = now

	return &stats, nil
}

func (db *DB) GetLocks(ctx context.Context) ([]LockInfo, error) {
	rows, err := db.conn.Query(ctx, `
		SELECT
			a.pid,
			COALESCE(a.usename, 'unknown'),
			l.locktype,
			l.granted,
			COALESCE(a.wait_event, 'none'),
			LEFT(COALESCE(a.query, ''), 60),
			COALESCE(c.relname, 'unknown')
		FROM pg_locks l
		JOIN pg_stat_activity a ON l.pid = a.pid
		LEFT JOIN pg_class c ON l.relation = c.oid
		WHERE a.query NOT LIKE '%pg_locks%'
		AND a.pid != pg_backend_pid()
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
			&l.Table,
		)
		if err != nil {
			continue
		}
		locks = append(locks, l)
	}
	return locks, nil
}
