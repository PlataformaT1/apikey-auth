package apikey

import "time"

type ApiKey struct {
	PlatformData map[string]interface{} `json:"platformData"`
	ExpiredAt    time.Time              `json:"expiredAt"`
	IsActive     bool                   `json:"isActive"`
	RequestCount int                    `json:"requestCount"`
	UsageLimits  UsageLimits            `json:"usageLimits"`
	ClientID     string                 `json:"clientId"`
}

type ApiKeys []ApiKey

type UsageLimits struct {
	DailyLimit int `json:"dailyLimit"`
	Limit      int `json:"limit"`
}
