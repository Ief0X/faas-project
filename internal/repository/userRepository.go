package repository

import (
	"faas-project/internal/message"
	"faas-project/internal/models"

	"github.com/nats-io/nats.go"
)

type UserRepository interface {
	CreateUser(user models.User) error
	GetByUsername(username string) (models.User, error)
}
type NATSUserRepository struct {
	js nats.JetStreamContext
}

func NewNATSUserRepository(js nats.JetStreamContext) *NATSUserRepository {
	return &NATSUserRepository{js: js}
}

// GetByUsername implements UserRepository.
func (r *NATSUserRepository) CreateUser(user models.User) error {
	kv, err := r.js.KeyValue("users")
	if err != nil {
		return err
	}
	_, err = kv.Put(user.Username, []byte(user.Password))
	if err != nil {
		return err
	}
	return nil
}

func (r *NATSUserRepository) GetByUsername(username string) (models.User, error) {
	kv, err := r.js.KeyValue("users")
	if err != nil {
		return models.User{}, err
	}
	entry, err := kv.Get(username)
	if err != nil {
		return models.User{}, err
	}
	var user models.User
	user.Username = username
	user.Password = string(entry.Value())

	return user, nil

}

func GetUserRepository() *NATSUserRepository {
	js := message.GetJetStream()
	return NewNATSUserRepository(js)
}
