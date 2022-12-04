package gopool

import (
	"context"
	"github.com/Senhnn/GoroutinePool"
)

func Go(f func()) {
	GoroutinePool.Go(f)
}

func CtxGo(ctx context.Context, f func()) {
	GoroutinePool.CtxGo(ctx, f)
}
