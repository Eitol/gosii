package sii

type Client interface {
	GetNameByRUT(rut string) (*Citizen, error)
}

type siiHTTPClient struct {
}

func NewClient() Client {
	return &siiHTTPClient{}
}
