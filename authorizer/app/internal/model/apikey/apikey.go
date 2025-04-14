package apikey

import (
	"time"
)

// ApiKey structure for API key authentication
type ApiKey struct {
	ID                 string                 `json:"id" bson:"_id,omitempty"`
	Name               string                 `json:"name" bson:"name"`
	ClientID           string                 `json:"clientId" bson:"clientId"`
	ApiKey             string                 `json:"apiKey" bson:"apiKey"`
	Platform           string                 `json:"platform" bson:"platform"`
	CreatedAt          time.Time              `json:"createdAt" bson:"createdAt"`
	ExpiredAt          time.Time              `json:"expiredAt" bson:"expiredAt"`
	UpdatedAt          time.Time              `json:"updatedAt" bson:"updatedAt"`
	CreatedAtLocalized string                 `json:"createdAtLocalized" bson:"createdAtLocalized"`
	ExpiredAtLocalized string                 `json:"expiredAtLocalized" bson:"expiredAtLocalized"`
	UpdatedAtLocalized string                 `json:"updatedAtLocalized" bson:"updatedAtLocalized"`
	IsActive           bool                   `json:"isActive" bson:"isActive"`
	RequestCount       int                    `json:"requestCount" bson:"requestCount"`
	PlatformData       map[string]interface{} `json:"platformData" bson:"platformData"`
	UsageLimits        UsageLimits            `json:"usageLimits" bson:"usageLimits"`
}

// UsageLimits defines the limits for API usage
type UsageLimits struct {
	DailyLimit        int    `json:"dailyLimit" bson:"dailyLimit"`
	Limit             int    `json:"limit" bson:"limit"`
	RequestsPerSecond int    `json:"requestsPerSecond" bson:"requestsPerSecond"`
	MaxPayloadSize    int    `json:"maxPayloadSize" bson:"maxPayloadSize"` // En MB
	PlanType          string `json:"planType" bson:"planType"`
}
