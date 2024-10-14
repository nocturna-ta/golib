# Transaction Manager

Supported driver:
- sql
- mongodb


## Usage

### Transaction Manager for Sql
Everytime before executing to database, you need to check tx to context. If tx exist in context you must use it to enable transaction.
```go
import (
	"context"
	"github.com/scprimesolution/golib/database/sql"
	"github.com/scprimesolution/golib/log"
	"github.com/scprimesolution/golib/txmanager"
	txSql "github.com/scprimesolution/golib/txmanager/sql"
	"github.com/scprimesolution/golib/txmanager/utils"
)

func main() {
	store := sql.New(sql.DBConfig{
		SlaveDSN:      "root:user123@tcp(localhost:3306)/test-db",
		MasterDSN:     "root:user123@tcp(localhost:3306)/test-db",
		RetryInterval: 10,
	}, sql.DriverMySQL)

	txMgr, err := txmanager.New(context.Background(), &txmanager.DriverConfig{
		Type:   "sql",
		Config: txSql.Config{DB: store},
	})
	if err != nil {
		log.Fatal("failed to instantiate txmanager")
	}

	// process
	findFn := func(ctx context.Context) error {
		// every call must check context if have any existing transaction
		tx := utils.GetSqlTx(ctx)

		var err error

		if tx != nil {
			// execute with transaction
			_, err = tx.ExecContext(ctx, `Select * from testt`)
		} else {
			// execute without transaction
			_, err = store.GetMaster().ExecContext(ctx, `Select * from testt`)
		}

		if err != nil {
			return err
		}
		return nil
	}

	transaction := func(ctx context.Context) (interface{}, error) {

		// add your transaction process here
		err := findFn(ctx)

		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	res, err := txMgr.Execute(context.Background(), transaction, nil)
	if err != nil {
		log.Fatal("failed transaction")
	}
}

```

### Transaction Manager for Mongodb

Required to import this `"github.com/scprimesolution/golib/txmanager/mongodb"` for instantiation of txmanager registry

```go
import (
	"context"
	"github.com/scprimesolution/golib/database/nosql/mongodb"
	"github.com/scprimesolution/golib/log"
	"github.com/scprimesolution/golib/txmanager"
	txMongo "github.com/scprimesolution/golib/txmanager/mongodb"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	client := mongodb.New(&mongodb.Config{
		URI:     "mongodb://localhost:27017",
		DB:      "test",
		AppName: "test",
	})

	db, err := client.GetDatabase()
	if err != nil {
		log.Fatal("failed to get mongo database")
	}

	txMgr, er := txmanager.New(context.Background(), &txmanager.DriverConfig{
		Type:   "mongodb",
		Config: txMongo.Config{DB: db},
	})
	if er != nil {
		log.Fatal("failed to instantiate tx manager")
	}

	transaction := func(ctx context.Context) (interface{}, error) {

		// add your transaction process here
		cur, err := db.Collection("test").Find(ctx, bson.D{})
		if err != nil {
			return nil, err
		}

		return cur, nil
	}

	res, err := txMgr.Execute(context.Background(), transaction, nil)
	if err != nil {
		log.Fatal("failed transaction")
	}
}
```