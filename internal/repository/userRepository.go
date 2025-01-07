package repository

import (
	"faas-project/internal/models"

	"github.com/nats-io/nats.go"
)

type UserRepository interface {
	CreateUser(user models.User) error
	GetByUsername(username string) (models.User, error)
}
type NatsUserRepository struct {
	js nats.JetStream
}


func (n *NatsUserRepository) CreateUser(user models.User) error {

}

// GetByUsername implements UserRepository.
func (n *NatsUserRepository) GetByUsername(username string) (models.User, error) {
	panic("unimplemented")
}

func GetUserRepository(js *nats.JetStream) UserRepository {
	return &NatsUserRepository{*js}
}
