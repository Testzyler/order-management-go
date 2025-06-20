package models

type ListInput struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

// make generic type with `Data` field as a slice of any type
type ListPaginated[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Size       int `json:"size"`
	TotalPages int `json:"total_pages"`
}
