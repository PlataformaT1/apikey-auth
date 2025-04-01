package company

import "time"

type Company struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	Phone       string    `json:"phone"`
	Mail        string    `json:"mail"`
	Rfc         string    `json:"rfc"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

type Companies []Company
