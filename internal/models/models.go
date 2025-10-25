// Package models contains data type
package models

// Request type
type Request struct {
	URL string `json:"url"`
}

// Response type
type Response struct {
	Result string `json:"result"`
}

// RequestBatchItem item of RequestBatch
type RequestBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// RequestBatch type for batch request
type RequestBatch []RequestBatchItem

// ResponseBatchItem item of ResponseBatch
type ResponseBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// ResponseBatch type for return value of GenerateShortURLBatch
type ResponseBatch []ResponseBatchItem

// ResponseUserItem item of ResponseUser
type ResponseUserItem struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

// ResponseUser type for return value GetUserURLS
type ResponseUser []ResponseUserItem

// RequestForDeleteURLS type for delete urls
type RequestForDeleteURLS []string

// TODO добавить тесты
