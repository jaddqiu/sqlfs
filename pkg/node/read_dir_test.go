package node

import (
	"context"
	"fmt"
	"sql-fs/pkg/mysql"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func ExampleReaddir() {
	f := func() sqlx.SqlConn {
		return mysql.New(
			"127.0.0.1",
			3306,
			"sqlfs",
			"123456",
			"sqlfs",
		)
	}
	root := &FileNode{
		PK:      0,
		NewConn: f,
	}

	fs, errno := readdir(context.Background(), root)
	if errno != 0 {
		fmt.Println("readdir error")
	}
	fmt.Println(fs.HasNext())
	for fs.HasNext() {
		f, err := fs.Next()
		if err != 0 {
			fmt.Printf("fs next derr: %v\n", err)
			return
		}
		fmt.Println(f)
	}
	fmt.Println("readdir success")

	// Output:
	// false
	// readdir success

}
