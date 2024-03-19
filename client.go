package gosii

type RequestMetadata struct {
	TotalCount int     `json:"total_count"`
	AvgTime    float64 `json:"avg_time"`
	Attempts   int     `json:"attempts"`
}

type Client interface {
	GetNameByRUT(rut string) (*Citizen, *RequestMetadata, error)
}
