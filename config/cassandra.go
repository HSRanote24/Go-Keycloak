package config

import (
	"log"

	"github.com/gocql/gocql"
)

var Session *gocql.Session

func InitCassandra() {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "testkeyspace"
	cluster.Consistency = gocql.Quorum

	var err error
	Session, err = cluster.CreateSession()
	if err != nil {
		log.Fatal("Cassandra connection failed:", err)
	}

	log.Println("Connected to Cassandra")
}
