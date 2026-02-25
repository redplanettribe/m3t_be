package helpers

import (
	"net/http"
	"strconv"

	"multitrackticketing/internal/domain"
)

// Pagination query parameter defaults and limits.
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// ParsePagination reads page and page_size from the request query string,
// clamps them to valid ranges, and returns domain.PaginationParams.
// Invalid or missing values fall back to defaults.
func ParsePagination(r *http.Request) domain.PaginationParams {
	page := DefaultPage
	if s := r.URL.Query().Get("page"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 1 {
			page = v
		}
	}
	pageSize := DefaultPageSize
	if s := r.URL.Query().Get("page_size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 1 {
			pageSize = v
			if pageSize > MaxPageSize {
				pageSize = MaxPageSize
			}
		}
	}
	return domain.PaginationParams{Page: page, PageSize: pageSize}
}

// PaginationMeta is the pagination metadata included in paginated list responses.
// swagger:model PaginationMeta
type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPaginationMeta builds PaginationMeta from the current page, page size, and total count.
// TotalPages is computed as ceiling(total / pageSize); if pageSize is 0, TotalPages is 0.
func NewPaginationMeta(page, pageSize, total int) PaginationMeta {
	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}
