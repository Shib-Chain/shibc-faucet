package server

import "time"

type Requester struct {
	Addr      string    `db:"addr"`
	IP        string    `db:"ip"`
	CreatedAt time.Time `db:"created_at"`
}
