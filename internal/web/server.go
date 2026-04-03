package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"gold_monitor/internal/advice"
	"gold_monitor/internal/dashboard"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	Service           *dashboard.Service
	DefaultInstrument string
	DefaultPosition   advice.Position
}

func (s *Server) Handler() (http.Handler, error) {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("sub static fs: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(sub))
	mux.HandleFunc("/api/dashboard", s.handleDashboard)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux, nil
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if s.Service == nil {
		writeJSONError(w, http.StatusInternalServerError, "dashboard service is not configured")
		return
	}

	query := r.URL.Query()
	instrument := query.Get("instrument")
	if instrument == "" {
		instrument = s.DefaultInstrument
	}
	if instrument == "" {
		instrument = "Au99.99"
	}

	pos := s.DefaultPosition
	if values, ok := query["cost"]; ok {
		if len(values) == 0 || values[0] == "" {
			pos.CostPerGram = 0
		} else {
			cost, err := strconv.ParseFloat(values[0], 64)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid cost")
				return
			}
			pos.CostPerGram = cost
		}
	}
	if values, ok := query["grams"]; ok {
		if len(values) == 0 || values[0] == "" {
			pos.Grams = 0
		} else {
			grams, err := strconv.ParseFloat(values[0], 64)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid grams")
				return
			}
			pos.Grams = grams
		}
	}
	if value := query.Get("sell_fee"); value != "" {
		fee, err := strconv.ParseFloat(value, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid sell_fee")
			return
		}
		pos.SellFeeRate = fee
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	resp, err := s.Service.Build(ctx, instrument, pos)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
