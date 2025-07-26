package models

type Request struct {
	URL string `json:"url"`
}

type Response struct {
	Result string `json:"result"`
}

type RequestBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}
type RequestBatch []RequestBatchItem

type ResponseBatchItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}
type ResponseBatch []ResponseBatchItem

type ResponseUserItem struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}
type ResponseUser []ResponseUserItem

type RequestForDeleteURLS []string

// TODO добавить тесты
