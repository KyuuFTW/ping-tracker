package main

import (
	"fmt"
	"time"

	"ping-tracker/tracker"
)

type InitialStateDTO struct {
	Filter      string `json:"filter"`
	IntervalMS  int    `json:"intervalMs"`
	PingEnabled bool   `json:"pingEnabled"`
	Paused      bool   `json:"paused"`
	Ready       bool   `json:"ready"`
	Scanning    bool   `json:"scanning"`
	LastScan    int64  `json:"lastScan"`
	LastError   string `json:"lastError"`
}

type StatusDTO struct {
	Ready     bool   `json:"ready"`
	Scanning  bool   `json:"scanning"`
	LastScan  int64  `json:"lastScan"`
	LastError string `json:"lastError"`
}

type ConnectionDTO struct {
	Key       string  `json:"key"`
	PID       int     `json:"pid"`
	AppName   string  `json:"appName"`
	Protocol  string  `json:"protocol"`
	Direction string  `json:"direction"`
	Local     string  `json:"local"`
	Remote    string  `json:"remote"`
	State     string  `json:"state"`
	PingMS    float64 `json:"pingMs"`
	Loss      float64 `json:"loss"`
	TxRate    string  `json:"txRate"`
	RxRate    string  `json:"rxRate"`
	Age       string  `json:"age"`
	Paused    bool    `json:"paused"`
}

type PingSampleDTO struct {
	Time  int64   `json:"time"`
	RTTMS float64 `json:"rttMs"`
	Loss  float64 `json:"loss"`
}

func connectionDTO(c *tracker.Connection, paused bool) ConnectionDTO {
	return ConnectionDTO{
		Key:       c.Key(),
		PID:       c.PID,
		AppName:   c.AppName,
		Protocol:  c.Protocol,
		Direction: string(c.Direction),
		Local:     endpoint(c.LocalAddr, c.LocalPort),
		Remote:    endpoint(c.RemoteAddr, c.RemotePort),
		State:     string(c.State),
		PingMS:    durationMS(c.Ping),
		Loss:      c.Loss,
		TxRate:    tracker.FormatBytes(c.TxRate),
		RxRate:    tracker.FormatBytes(c.RxRate),
		Age:       c.ConnAge.Truncate(time.Second).String(),
		Paused:    paused,
	}
}

func endpoint(addr string, port int) string {
	if port <= 0 {
		return addr
	}
	return fmt.Sprintf("%s:%d", addr, port)
}

func durationMS(d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(d.Microseconds()) / 1000.0
}
