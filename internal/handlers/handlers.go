package handlers

import "lorem-backend/internal/repositories"

type Handlers struct {
	*repositories.Repositories
}

func NewHandlers(repos *repositories.Repositories) *Handlers {
	return &Handlers{Repositories: repos}
}
