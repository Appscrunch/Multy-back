package store

type InMemoryStore struct {
	store map[string]string
}

func InitInMemoryStore() (DataStore, error) {
	return &InMemoryStore{
		store: make(map[string]string),
	}, nil
}

func (i *InMemoryStore) Close() error {
	return nil
}
