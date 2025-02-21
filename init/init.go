package init

import "context"

func init() {
	ctx := context.Background()
	initDB(ctx)
	initMinIO(ctx)
}
