package api

import (
	"time"

	"github.com/AdmiralBulldogTv/VodApi/src/global"
	"github.com/AdmiralBulldogTv/VodApi/src/utils"
	jsoniter "github.com/json-iterator/go"

	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func New(gCtx global.Context) <-chan struct{} {
	done := make(chan struct{})
	gql := GqlHandler(gCtx)
	webhookTwitch := WebhookTwitchHandler(gCtx)

	server := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			defer func() {
				gCtx.Inst().Prometheus.ResponseTimeMilliseconds().Observe(float64(time.Since(start)/time.Microsecond) / 1000)
				l := logrus.WithFields(logrus.Fields{
					"status":     ctx.Response.StatusCode(),
					"duration":   time.Since(start) / time.Millisecond,
					"entrypoint": "api",
					"path":       utils.B2S(ctx.Path()),
				})
				if err := recover(); err != nil {
					l.Error("panic in handler: ", err)
				} else {
					l.Info("")
				}
			}()

			path := utils.B2S(ctx.Path())

			if path == "/gql" {
				gql(ctx)
			} else if path == "/twitch/webhook" {
				webhookTwitch(ctx)
			} else {
				ctx.SetStatusCode(fasthttp.StatusNotFound)
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
