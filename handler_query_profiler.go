package main

import (
	"net/http"
)

// handleQueryProfilerStats returns query profiling statistics
func handleQueryProfilerStats(w http.ResponseWriter, r *http.Request) {
	if profiler == nil {
		jsonErr(w, "Query profiler not initialized", 500)
		return
	}
	
	stats := profiler.GetStats()
	jsonResp(w, stats)
}

// handleQueryProfilerSlowQueries returns slow queries
func handleQueryProfilerSlowQueries(w http.ResponseWriter, r *http.Request) {
	if profiler == nil {
		jsonErr(w, "Query profiler not initialized", 500)
		return
	}
	
	slow := profiler.GetSlowQueries()
	jsonResp(w, map[string]interface{}{
		"slow_queries": slow,
		"count":        len(slow),
		"threshold":    profiler.slowThreshold.String(),
	})
}

// handleQueryProfilerAllQueries returns all recorded queries
func handleQueryProfilerAllQueries(w http.ResponseWriter, r *http.Request) {
	if profiler == nil {
		jsonErr(w, "Query profiler not initialized", 500)
		return
	}
	
	queries := profiler.GetAllQueries()
	jsonResp(w, map[string]interface{}{
		"queries": queries,
		"count":   len(queries),
	})
}

// handleQueryProfilerReset resets the profiler
func handleQueryProfilerReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonErr(w, "method not allowed", 405)
		return
	}
	
	if profiler == nil {
		jsonErr(w, "Query profiler not initialized", 500)
		return
	}
	
	profiler.Reset()
	jsonResp(w, map[string]interface{}{"message": "Profiler reset successfully"})
}
