package components

import (
	"fmt"

	"github.com/doraemonkeys/brick"
)

// UserService implements Brick and provides user-related services.
type UserService struct {
	DB *Database `brick:"Database"`
}

// BrickTypeID implements the Brick interface.
func (*UserService) BrickTypeID() string {
	return "UserService"
}

// NewBrick implements the BrickNewer interface.
func (us *UserService) NewBrick(jsonConfig []byte) brick.Brick {
	return &UserService{}
}

// GetUser retrieves a user by ID (simulated).
func (us *UserService) GetUser(id int) {
	fmt.Printf("Retrieving user with ID: %d\n", id)
	us.DB.Connect()
}
