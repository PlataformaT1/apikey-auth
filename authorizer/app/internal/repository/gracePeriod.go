package mongodb

import (
	"apikey/internal/model/gracePeriod"
	"context"
)

type GracePeriodRepository interface {
	ValidationGracePeriod(ctx context.Context) (*gracePeriod.CommerceRule, error)
}

