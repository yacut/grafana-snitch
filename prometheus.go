package main

import "github.com/prometheus/client_golang/prometheus"

func prometheusInit() {
	promSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grafana_snitch_success",
			Help: "Cumulative number of role update operations",
		},
		[]string{"count"},
	)

	promErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grafana_snitch_errors",
			Help: "Cumulative number of errors during role update operations",
		},
		[]string{"count"},
	)
	prometheus.MustRegister(promSuccess)
	prometheus.MustRegister(promErrors)
}
