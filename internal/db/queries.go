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

// New creates a new database connection pool and returns a DB instance.
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

// Close closes the database connection pool.
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
	LastVacuum      string
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

// GetOverviewStats retrieves general overview statistics for the database.
func (db *DB) GetOverviewStats(ctx context.Context) (*OverviewStats, error) {
	var stats OverviewStats

	err := db.conn.QueryRow(ctx, `
		WITH activity AS (
			SELECT 
				COUNT(*) FILTER (WHERE state = 'active') as active_conns,
				COUNT(*) FILTER (WHERE state = 'idle') as idle_conns
			FROM pg_stat_activity
		),
		db_stats AS (
			SELECT
				ROUND(
					SUM(blks_hit) * 100.0 / NULLIF(SUM(blks_hit) + SUM(blks_read), 0),
					2
				) as cache_hit_ratio,
				SUM(xact_commit + xact_rollback) as total_xact
			FROM pg_stat_database
			WHERE datname = CURRENT_DATABASE()
			GROUP BY datname
		)
		SELECT
			CURRENT_DATABASE(),
			SPLIT_PART(VERSION(), ' ', 2),
			PG_SIZE_PRETTY(PG_DATABASE_SIZE(CURRENT_DATABASE())),
			COALESCE(activity.active_conns, 0),
			COALESCE(activity.idle_conns, 0),
			CURRENT_SETTING('max_connections')::int,
			DATE_TRUNC('second', NOW() - PG_POSTMASTER_START_TIME())::text,
			COALESCE(db_stats.cache_hit_ratio, 0.0),
			COALESCE(db_stats.total_xact, 0)
		FROM activity, db_stats
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

// GetActiveQueries retrieves a list of currently active queries.
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

// GetWALStats retrieves Write-Ahead Logging statistics.
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

var lastVacuum string
err = db.conn.QueryRow(ctx, `
    SELECT 
        COALESCE(
            date_trunc('second', now() - max(last_autovacuum))::text,
            'never'
        )
    FROM pg_stat_user_tables
`).Scan(&lastVacuum)
if err != nil || lastVacuum == "" {
    lastVacuum = "never"
}
stats.LastVacuum = lastVacuum

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

// GetLocks retrieves information about current database locks.
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
		AND a.query NOT LIKE '%pg_stat%'
		AND l.granted = true
		AND a.state = 'active'
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
