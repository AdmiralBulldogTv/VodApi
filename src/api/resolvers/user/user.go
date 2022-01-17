package user

import (
	"context"
	"time"

	"github.com/AdmiralBulldogTv/VodApi/graph/generated"
	"github.com/AdmiralBulldogTv/VodApi/graph/model"
	"github.com/AdmiralBulldogTv/VodApi/src/api/helpers"
	"github.com/AdmiralBulldogTv/VodApi/src/api/types"
	"github.com/AdmiralBulldogTv/VodApi/src/structures"
	"github.com/AdmiralBulldogTv/VodApi/src/svc/mongo"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Resolver struct {
	types.Resolver
}

func New(r types.Resolver) generated.UserResolver {
	return &Resolver{
		Resolver: r,
	}
}

func (r *Resolver) Vods(ctx context.Context, obj *model.User, limit int, page int, search *string, after *time.Time, before *time.Time) ([]*model.Vod, error) {
	filter := bson.M{
		"user_id": obj.ID,
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	if page < 0 {
		page = 0
	}

	if search != nil {
		filter["$text"] = bson.M{
			"$search": *search,
		}
	}

	timeFilter := bson.M{}

	if after != nil {
		timeFilter["$gte"] = *after
	}

	if before != nil {
		timeFilter["$lte"] = *before
	}

	if len(timeFilter) != 0 {
		filter["created_at"] = timeFilter
	}

	cur, err := r.Ctx.Inst().Mongo.Collection(mongo.CollectionNameVods).Find(ctx, filter, options.Find().SetLimit(int64(limit)).SetSkip(int64(page)*int64(limit)))
	dbVods := []structures.Vod{}
	if err == nil {
		err = cur.All(ctx, &dbVods)
	}
	if err != nil {
		logrus.Error("failed to fetch vods: ", err)
		return nil, helpers.ErrInternalServerError
	}

	vods := make([]*model.Vod, len(dbVods))
	for i, vod := range dbVods {
		vods[i] = vod.ToModel()
	}

	return vods, nil
}
