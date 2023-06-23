package gosii

type Client interface {
	GetNameByRUT(rut string) (*Citizen, error)
}
