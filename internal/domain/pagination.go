package domain

// PaginationParams holds offset-based pagination parameters for list queries.
type PaginationParams struct {
	Page     int
	PageSize int
}

// Offset returns the row offset for the current page (0-based).
// Formula: (Page - 1) * PageSize.
func (p PaginationParams) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}
