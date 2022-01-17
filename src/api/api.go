package api

import (
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
	jsoniter "github.com/json-iterator/go"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})
	gql := GqlHandler(gCtx)

	server := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			defer func() {
				gCtx.Inst().Prometheus.ResponseTimeMilliseconds().Observe(float64(time.Since(start)/time.Microsecond) / 1000)
				l := logrus.WithFields(logrus.Fields{
					"status":     ctx.Response.StatusCode(),
					"duration":   time.Since(start) / time.Millisecond,
					"entrypoint": "api",
				})
				if err := recover(); err != nil {
					l.Error("panic in handler: ", err)
				} else {
					l.Info("")
				}
			}()

			switch utils.B2S(ctx.Path()) {
			case "/gql":
				gql(ctx)
			default:
				ctx.SetStatusCode(404)
			}
		},
		ReadTimeout:     time.Second * 10,
		WriteTimeout:    time.Second * 10,
		CloseOnShutdown: true,
	}

	go func() {
		if err := server.ListenAndServe(gCtx.Config().API.Bind); err != nil {
			logrus.Fatal("failed to start api server: ", err)
		}
	}()

	go func() {
		<-gCtx.Done()
		_ = server.Shutdown()
		close(done)
	}()

	return done
}

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
		switch utils.B2S(ctx.Method()) {
		case "GET":
			query, _ := url.ParseQuery(ctx.QueryArgs().String())
			if err := qstring.Unmarshal(query, &req); err != nil {
				ctx.SetStatusCode(400)
				return
			}
		case "POST":
			if err := json.Unmarshal(ctx.Request.Body(), &req); err != nil {
				ctx.SetStatusCode(400)
				return
			}
		default:
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
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
		data, _ := json.Marshal(result)
		ctx.SetBody(data)
	}
}
