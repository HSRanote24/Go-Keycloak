package services

import (
	"go-keycloack/config"
	"go-keycloack/models"

	"github.com/gocql/gocql"
)

func GetUserByID(id gocql.UUID) (*models.User, error) {
	var u models.User
	err := config.Session.Query(
		"SELECT id, username, email, firstname, lastname FROM users WHERE id = ?",
		id,
	).Consistency(gocql.One).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByUsername(username string) (*models.User, error) {
	var u models.User
	err := config.Session.Query(
		"SELECT id, username, email, firstname, lastname FROM users WHERE username = ?",
		username,
	).Consistency(gocql.One).Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func CreateUser(user *models.User) error {
	user.ID = gocql.TimeUUID()
	return config.Session.Query(
		"INSERT INTO users (id, username, email, firstname, lastname) VALUES (?, ?, ?, ?, ?)",
		user.ID, user.Username, user.Email, user.FirstName, user.LastName,
	).Exec()
}

func UpdateUser(id gocql.UUID, user *models.User) error {
	return config.Session.Query(
		"UPDATE users SET username = ?, email = ?, firstname = ?, lastname = ? WHERE id = ?",
		user.Username, user.Email, user.FirstName, user.LastName, id,
	).Exec()
}

func DeleteUser(id gocql.UUID) error {
	return config.Session.Query("DELETE FROM users WHERE id = ?", id).Exec()
}

// GetAllUsers fetches all users from the database
func GetAllUsers() ([]models.User, error) {
	var users []models.User
	iter := config.Session.Query("SELECT id, username, email, firstname, lastname FROM users").Iter()
	var u models.User
	for iter.Scan(&u.ID, &u.Username, &u.Email, &u.FirstName, &u.LastName) {
		users = append(users, u)
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	return users, nil
}
