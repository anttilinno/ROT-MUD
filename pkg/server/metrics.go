package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the server
type Metrics struct {
	PlayersOnline   prometheus.Gauge
	NPCsActive      prometheus.Gauge
	CommandsTotal   prometheus.Counter
	CombatDamage    prometheus.Histogram
	PulseLatency    prometheus.Histogram
	SpellsCast      prometheus.Counter
	ConnectionsOpen prometheus.Gauge
}

// NewMetrics creates and registers all metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		PlayersOnline: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rotmud_players_online",
			Help: "Number of players currently online",
		}),
		NPCsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rotmud_npcs_active",
			Help: "Number of NPCs currently active in the world",
		}),
		CommandsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rotmud_commands_total",
			Help: "Total number of commands processed",
		}),
		CombatDamage: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "rotmud_combat_damage",
			Help:    "Distribution of combat damage dealt",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		PulseLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "rotmud_pulse_latency_seconds",
			Help:    "Game loop pulse processing latency",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		}),
		SpellsCast: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "rotmud_spells_cast_total",
			Help: "Total number of spells cast",
		}),
		ConnectionsOpen: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "rotmud_connections_open",
			Help: "Number of open connections (telnet + websocket)",
		}),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.PlayersOnline,
		m.NPCsActive,
		m.CommandsTotal,
		m.CombatDamage,
		m.PulseLatency,
		m.SpellsCast,
		m.ConnectionsOpen,
	)

	return m
}

// Handler returns the Prometheus HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// UpdateCounts updates player and NPC counts
func (m *Metrics) UpdateCounts(players, npcs int) {
	m.PlayersOnline.Set(float64(players))
	m.NPCsActive.Set(float64(npcs))
}

// IncrementCommands increments the commands counter
func (m *Metrics) IncrementCommands() {
	m.CommandsTotal.Inc()
}

// RecordDamage records a damage value
func (m *Metrics) RecordDamage(damage int) {
	m.CombatDamage.Observe(float64(damage))
}

// RecordPulseLatency records pulse processing time
func (m *Metrics) RecordPulseLatency(seconds float64) {
	m.PulseLatency.Observe(seconds)
}

// IncrementSpellsCast increments the spells cast counter
func (m *Metrics) IncrementSpellsCast() {
	m.SpellsCast.Inc()
}

// SetConnections sets the current connection count
func (m *Metrics) SetConnections(count int) {
	m.ConnectionsOpen.Set(float64(count))
}
