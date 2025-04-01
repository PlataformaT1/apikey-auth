package gracePeriod

type CommerceRule struct {
	IdCommerce       string                 `json:"id_commerce" bson:"id_commerce"`
	MaxAmount        float32                `json:"max_amount" bson:"max_amount"`
	LimitTransaction int                    `json:"limit_transaction" bson:"limit_transaction"`
	StatusCommerce   map[string]interface{} `json:"status_commerce" bson:"status_commerce"`
	Category		string                   `json:"category" bson:"category"`
	CreatedAt       string                   `json:"createdAt" bson:"CreatedAt"`
}

type CommerceRules []*CommerceRule