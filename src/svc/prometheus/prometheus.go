package prometheus

import (
	"github.com/AdmiralBulldogTv/VodApi/src/configure"
	"github.com/AdmiralBulldogTv/VodApi/src/instance"

	"github.com/prometheus/client_golang/prometheus"
)

type mon struct {
	responseTimeMilliseconds prometheus.Histogram
	twitchChatMessages       prometheus.Histogram
}

func (m *mon) Register(r prometheus.Registerer) {
	r.MustRegister(
		m.responseTimeMilliseconds,
		m.twitchChatMessages,
	)
}

func (m *mon) ResponseTimeMilliseconds() prometheus.Histogram {
	return m.responseTimeMilliseconds
}

func (m *mon) TwitchChatMessages() prometheus.Histogram {
	return m.twitchChatMessages
}

func LabelsFromKeyValue(kv []configure.KeyValue) prometheus.Labels {
	mp := prometheus.Labels{}

	for _, v := range kv {
		mp[v.Key] = v.Value
	}

	return mp
}

func New(opts SetupOptions) instance.Prometheus {
	return &mon{
		responseTimeMilliseconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "api_response_time_milliseconds",
			Help: "The response time in milliseconds",
		}),
		twitchChatMessages: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "api_twitch_chat_messages",
			Help: "The number of messages read",
		}),
	}
}

type SetupOptions struct {
	Labels prometheus.Labels
}
