// Package response defines common STKCNSL API response envelope types.
package response

// Envelope is the standard STKCNSL API response wrapper for list endpoints.
type Envelope[T any] struct {
	Status       string  `json:"status"`
	Message      string  `json:"message"`
	Timezone     string  `json:"timezone"`
	CurrentPage  int     `json:"current_page"`
	Data         []T     `json:"data"`
	FirstPageURL *string `json:"first_page_url"`
	From         *int    `json:"from"`
	LastPage     int     `json:"last_page"`
	LastPageURL  *string `json:"last_page_url"`
	NextPageURL  *string `json:"next_page_url"`
	Path         string  `json:"path"`
	PerPage      int     `json:"per_page"`
	PrevPageURL  *string `json:"prev_page_url"`
	To           *int    `json:"to"`
	Total        int     `json:"total"`
}

// Single is for endpoints that return a single object in data.
type Single[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}
