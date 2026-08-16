// Package manager 实现检查项目先后关系的权限、校验、传递推导和循环防护。
package manager

import (
	"errors"

	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type Manager struct {
	store    precedence.Store
	projects precedence.ProjectDirectory
}

func New(store precedence.Store, projects precedence.ProjectDirectory) (*Manager, error) {
	if store == nil {
		return nil, errors.New("precedence store is required")
	}
	if projects == nil {
		return nil, errors.New("precedence project directory is required")
	}
	return &Manager{store: store, projects: projects}, nil
}
