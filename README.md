LiteFS Go Library [![Go Reference](https://pkg.go.dev/badge/github.com/superfly/litefs-go.svg)](https://pkg.go.dev/github.com/superfly/litefs-go)
=================

This Go library is for interacting with LiteFS features that cannot be accessed
through the typical SQLite API.


## Halting

LiteFS provides the ability to halt writes on the primary node in order that
replicas may execute writes remotely and forward them back to the primary. This
isn't necessary in most usage, however, it can make running migrations simpler.

Write forwarding from the replica is much slower than executing the write
transaction directly on the primary so only use this for migrations or low-write
scenarios.

```go
// Initialize the database.
db, err := sql.Open("sqlite3", "/litefs/my.db")
if err != nil {
	return err
}
defer db.Close()

// Execute a write transaction from any node.
// If this is a replica, it will run the inner function with the HALT lock.
if err := litefs.WithHalt("/litefs/my.db", func() error {
	_, err := db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name string)`)
	return err
}); err != nil {
	return err
}
```
