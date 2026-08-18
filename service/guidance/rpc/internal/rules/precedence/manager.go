package precedence

import "errors"

type Manager struct {
	store    Store
	projects ProjectDirectory
}

func New(store Store, projects ProjectDirectory) (*Manager, error) {
	if store == nil {
		return nil, errors.New("precedence store is required")
	}
	if projects == nil {
		return nil, errors.New("precedence project directory is required")
	}
	return &Manager{store: store, projects: projects}, nil
}
