package resolvers

import (
	"github.com/AdmiralBulldogTv/VodApi/graph/generated"
	"github.com/AdmiralBulldogTv/VodApi/src/api/resolvers/query"
	"github.com/AdmiralBulldogTv/VodApi/src/api/resolvers/user"
	"github.com/AdmiralBulldogTv/VodApi/src/api/resolvers/vod"
	"github.com/AdmiralBulldogTv/VodApi/src/api/types"
)

type Resolver struct {
	types.Resolver
	query generated.QueryResolver
	vod   generated.VodResolver
	user  generated.UserResolver
}

func New(r types.Resolver) generated.ResolverRoot {
	return &Resolver{
		Resolver: r,
		query:    query.New(r),
		vod:      vod.New(r),
		user:     user.New(r),
	}
}

func (r *Resolver) Query() generated.QueryResolver {
	return r.query
}

func (r *Resolver) Vod() generated.VodResolver {
	return r.vod
}

func (r *Resolver) User() generated.UserResolver {
	return r.user
}
