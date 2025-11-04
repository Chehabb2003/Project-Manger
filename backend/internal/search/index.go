package search

type Index interface {
	Add(id string, text string) error
	Query(q string) ([]string, error)
}

type noop struct{}

func New() Index { return noop{} }

func (n noop) Add(string, string) error       { return nil }
func (n noop) Query(string) ([]string, error) { return nil, nil }
