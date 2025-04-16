package models

import (
	"github.com/gocql/gocql"
)

type User struct {
	ID        gocql.UUID `json:"id"`
	Username  string     `json:"username" validate:"required,min=3,max=32"`
	Email     string     `json:"email" validate:"required,email"`
	FirstName string     `json:"firstname"`
	LastName  string     `json:"lastname"`
}
