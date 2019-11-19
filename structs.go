package goUtils

import "github.com/prometheus/client_golang/prometheus"

type Utils struct {
	options *Options
}

type Options struct {
	EnvironmentVarPrefix     string
	EnvironmentVarSeparator  string
	SentryURL                string
	PurifyReplacer           string
	PrometheusRequestsVector *prometheus.HistogramVec
	PrometheusHandlersVector *prometheus.HistogramVec
	LanguageISOCode          string
}
type NameElements struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
