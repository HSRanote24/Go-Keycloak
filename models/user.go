package models

import "github.com/gocql/gocql"

type User struct {
	ID       gocql.UUID `json:"id"`
	Username string     `json:"username"`
	Email    string     `json:"email"`
	// Add more fields as needed
}
