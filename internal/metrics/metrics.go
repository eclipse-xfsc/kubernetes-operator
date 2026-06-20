package metrics

import "github.com/prometheus/client_golang/prometheus"

var ReconcileTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "xsfc_operator_reconcile_total", Help: "Total number of reconciliations by controller."}, []string{"controller"})
var InjectionRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "xsfc_operator_injection_requests_total", Help: "Total number of detected injection requests by type."}, []string{"type"})

func Register() {
	prometheus.MustRegister(ReconcileTotal)
	prometheus.MustRegister(InjectionRequestsTotal)
}
