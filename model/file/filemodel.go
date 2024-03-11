package file

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ FileModel = (*customFileModel)(nil)

type (
	// FileModel is an interface to be customized, add more methods here,
	// and implement the added methods in customFileModel.
	FileModel interface {
		fileModel
		withSession(session sqlx.Session) FileModel
		GetChildrenByParentId(ctx context.Context, parentId int64) ([]*File, error)
	}

	customFileModel struct {
		*defaultFileModel
	}
)

// NewFileModel returns a model for the database table.
func NewFileModel(conn sqlx.SqlConn) FileModel {
	return &customFileModel{
		defaultFileModel: newFileModel(conn),
	}
}

func (m *customFileModel) withSession(session sqlx.Session) FileModel {
	return NewFileModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customFileModel) GetChildrenByParentId(ctx context.Context, parentId int64) ([]*File, error) {
	query := fmt.Sprintf("select %s from %s where `parent_dir` = ?", fileRows, m.table)
	var resp []*File
	err := m.conn.QueryRowsCtx(ctx, &resp, query, parentId)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
