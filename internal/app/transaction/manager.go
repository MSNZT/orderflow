package transaction

import "context"

type Manager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
