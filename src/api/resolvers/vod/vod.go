package vod

import (
	"context"

	"github.com/AdmiralBulldogTv/VodApi/graph/generated"
	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"github.com/AdmiralBulldogTv/VodApi/src/api/loaders"
	"github.com/AdmiralBulldogTv/VodApi/src/api/types"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.VodResolver {
	return &Resolver{Resolver: r}
}

func (r *Resolver) User(ctx context.Context, obj *model.Vod) (*model.User, error) {
	return loaders.For(ctx).UserLoader.Load(obj.UserID)
}
