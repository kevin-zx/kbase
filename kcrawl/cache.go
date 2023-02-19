package kcrawl

import "time"

type cacheData struct {
	Request   request   `json:"request"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

type request struct {
	Method  string `json:"method"`
	URL     string `json:"url"`
	Payload string `json:"payload"`
}
