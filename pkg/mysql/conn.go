package mysql

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func New(host string, port int, user, password, db string) sqlx.SqlConn {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=false&autocommit=true&parseTime=true&loc=Local", user, password, host, port, db)
	return sqlx.NewSqlConn("mysql", dsn)
}
