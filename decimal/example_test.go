// Run Tarantool instance before example execution:
//
// Terminal 1:
// $ cd decimal
// $ TEST_TNT_LISTEN=3013 TEST_TNT_WORK_DIR=$(mktemp -d -t 'tarantool.XXX') tarantool config.lua
//
// Terminal 2:
// $ go test -v example_test.go
package decimal_test

import (
	"context"
	"log"
	"time"

	"github.com/tarantool/go-tarantool/v2"
	. "github.com/tarantool/go-tarantool/v2/decimal"
)

// To enable support of decimal in msgpack with
// https://github.com/shopspring/decimal,
// import tarantool/decimal submodule.
func Example() {
	server := "127.0.0.1:3013"
	opts := tarantool.Opts{
		Timeout: 5 * time.Second,
		User:    "test",
		Pass:    "test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	client, err := tarantool.Connect(ctx, server, opts)
	cancel()
	if err != nil {
		log.Fatalf("Failed to connect: %s", err.Error())
	}

	spaceNo := uint32(524)

	number, err := MakeDecimalFromString("-22.804")
	if err != nil {
		log.Fatalf("Failed to prepare test decimal: %s", err)
	}

	resp, err := client.Do(tarantool.NewReplaceRequest(spaceNo).
		Tuple([]interface{}{number}),
	).Get()
	if err != nil {
		log.Fatalf("Decimal replace failed: %s", err)
	}
	if resp == nil {
		log.Fatalf("Response is nil after Replace")
	}

	log.Println("Decimal tuple replace")
	log.Println("Error", err)
	log.Println("Code", resp.Code)
	log.Println("Data", resp.Data)
}
