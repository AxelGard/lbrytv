package storage

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/lbryio/lbry.go/v2/extras/crypto"
)

// CreateTestConn creates a temporary test database and returns a connection object for accessing it
// plus a cleanup callback that should be deferredly called by function caller for properly getting rid
// of this temporary database.
func CreateTestConn(params ConnParams) (*Connection, func()) {
	rand.Seed(time.Now().UnixNano())

	conn := InitConn(params)
	err := conn.Connect()
	if err != nil {
		panic(err)
	}

	tempDbName := crypto.RandString(24)
	params.DBName = tempDbName
	conn.CreateDB(params.DBName)

	testConn := InitConn(params)
	err = testConn.Connect()
	if err != nil {
		panic(fmt.Sprintf("test DB connection failed: %v", err))
	}
	testConn.MigrateUp()

	return testConn, func() {
		testConn.Close()
		UnsetDefaultConnection()
		conn.DropDB(tempDbName)
		conn.Close()
	}
}

func UnsetDefaultConnection() {
	Conn = nil
}
