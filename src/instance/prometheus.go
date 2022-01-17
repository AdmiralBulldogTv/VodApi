package instance

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus interface {
	Register(prometheus.Registerer)
	ResponseTimeMilliseconds() prometheus.Histogram
	TwitchChatMessages() prometheus.Histogram
}
