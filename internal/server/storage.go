package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Storage struct {
	db *sqlx.DB
}

type RequesterFilter struct {
	Addr string
	IP   string
}

func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{db: db}
}

func (st *Storage) CreateRequester(ctx context.Context, r *Requester) error {
	cmd := "INSERT INTO requesters (addr, ip, created_at) VALUES (:addr, :ip, :created_at)"
	_, err := st.db.NamedExecContext(ctx, cmd, r)
	if err != nil {
		return fmt.Errorf("insert requester failed: adrr:%s - ip:%s - detail:%w", r.Addr, r.IP, err)
	}

	return nil
}

func (st *Storage) GetRequester(ctx context.Context, filter RequesterFilter) (*Requester, error) {
	var r Requester
	cmd := "SELECT * FROM requesters WHERE addr = $1 OR ip = $2 LIMIT 1"
	err := st.db.GetContext(ctx, &r, cmd, filter.Addr, filter.IP)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &r, nil
}
