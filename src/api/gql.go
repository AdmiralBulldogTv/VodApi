package api

import (
	"bytes"
	"context"
	"net/url"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/AdmiralBulldogTv/VodApi/graph/generated"
	"github.com/AdmiralBulldogTv/VodApi/src/api/cache"
	"github.com/AdmiralBulldogTv/VodApi/src/api/complexity"
	"github.com/AdmiralBulldogTv/VodApi/src/api/helpers"
	"github.com/AdmiralBulldogTv/VodApi/src/api/loaders"
	"github.com/AdmiralBulldogTv/VodApi/src/api/middleware"
	"github.com/AdmiralBulldogTv/VodApi/src/api/resolvers"
	"github.com/AdmiralBulldogTv/VodApi/src/api/types"
	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/AdmiralBulldogTv/VodApi/src/svc/redis"
	"github.com/AdmiralBulldogTv/VodApi/src/utils"
	"github.com/dyninc/qstring"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type gqlRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operation_name"`
	RequestID     string                 `json:"request_id"`
}

func GqlHandler(gCtx global.Context) func(ctx *fasthttp.RequestCtx) {
	schema := NewWrapper(generated.NewExecutableSchema(generated.Config{
		Resolvers:  resolvers.New(types.Resolver{Ctx: gCtx}),
		Directives: middleware.New(gCtx),
		Complexity: complexity.New(gCtx),
	}))

	schema.Use(&extension.ComplexityLimit{
		Func: func(ctx context.Context, rc *graphql.OperationContext) int {
			// we can define limits here
			return 75
		},
	})

	schema.Use(extension.Introspection{})
	schema.Use(extension.AutomaticPersistedQuery{
		Cache: cache.NewRedisCache(gCtx, redis.RedisPrefix+":", time.Hour*6),
	})

	schema.SetRecoverFunc(func(ctx context.Context, err interface{}) (userMessage error) {
		logrus.Error("panic in handler: ", err)
		return helpers.ErrInternalServerError
	})

	loader := loaders.New(gCtx)

	return func(ctx *fasthttp.RequestCtx) {
		req := gqlRequest{}
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")
		ctx.Response.Header.Set("Access-Control-Max-Age", "86400")

		switch utils.B2S(ctx.Method()) {
		case "GET":
			query, _ := url.ParseQuery(ctx.QueryArgs().String())
			if err := qstring.Unmarshal(query, &req); err != nil {
				ctx.SetStatusCode(400)
				return
			}
			data, _ := json.Marshal(req)
			ctx.Request.SetBody(data)
		case "POST":
		case "OPTIONS":
			ctx.SetStatusCode(204)
			return
		default:
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			return
		}

		decoder := json.NewDecoder(bytes.NewReader(ctx.Request.Body()))
		decoder.UseNumber()
		if err := decoder.Decode(&req); err != nil {
			ctx.SetStatusCode(400)
			return
		}

		// Execute the query
		result := schema.Process(context.WithValue(ctx, loaders.LoadersKey, loader), graphql.RawParams{
			Query:         req.Query,
			OperationName: req.OperationName,
			Variables:     req.Variables,
		})

		ctx.SetStatusCode(result.Status)
		ctx.SetContentType("application/json")
		data, _ := json.Marshal(result.Response)
		ctx.SetBody(data)
	}
}
