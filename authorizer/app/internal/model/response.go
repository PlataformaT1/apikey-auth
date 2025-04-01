package model

import "time"

type Delete struct {
	Id      int64     `json:"id"`
	Deleted time.Time `json:"deleted_at"`
}

type BaseResponse struct {
	Status    string `json:"status"`
	Code      int    `json:"code"`
	Datetime  string `json:"datetime"`
	Timestamp int64  `json:"timestamp"`
}
