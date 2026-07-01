package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/internal/env"
	"gateway/internal/log"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storer interface {
	StoreLogs([]log.LogData) error
	ReadLogsHandler() http.Handler
}

type storage struct {
	ctx context.Context
	db  *pgxpool.Pool
}

func NewStorage(ctx context.Context) *storage {

	pool, err := pgxpool.New(ctx, env.GetDbConnString())
	if err != nil {
		panic(err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to PostgreSQL!")

	return &storage{
		ctx: ctx,
		db:  pool,
	}
}

func (s *storage) StoreLogs(logs []log.LogData) error {
	var rows [][]any = make([][]any, 0, len(logs))
	for _, l := range logs {
		rows = append(rows, []any{l.Data.ReqId, l.CreatedAt, l.Source, l.Data})
	}

	inserted, err := s.db.CopyFrom(s.ctx, pgx.Identifier{"request_logs"}, []string{"req_id", "created_at", "source", "data"}, pgx.CopyFromRows(rows))
	if err != nil {
		return err
	}
	if len(rows) > 0 && inserted == 0 {
		return errors.New("failed to insert rows for unknown reason")
	}
	return nil
}

func (s *storage) readLogs() (*[]log.LogData, error) {
	rows, err := s.db.Query(s.ctx, "SELECT created_at, source, data from request_logs ORDER BY created_at LIMIT 5000")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var logs []log.LogData
	for rows.Next() {
		var log log.LogData
		rows.Scan(&log.CreatedAt, &log.Source, &log.Data)
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &logs, nil

}

func (s *storage) ReadLogsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		logs, err := s.readLogs()

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			encoder.Encode(struct {
				message string
			}{
				message: "error occured while reading logs",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		encoder.Encode(*logs)
	})
}
