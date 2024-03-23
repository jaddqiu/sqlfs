package file

import (
	"context"
	"fmt"

	"github.com/hanwen/go-fuse/v2/fuse"
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

func (f *File) InodeId() uint64 {
	return uint64(f.Id + 10000)
}

func (f *File) Attr() fuse.Attr {
	var mode uint32
	if f.Type == "directory" {
		mode = fuse.S_IFDIR
	} else {
		mode = fuse.S_IFREG
	}
	return fuse.Attr{
		Ino:   f.InodeId(),
		Size:  uint64(len(f.Content.String)),
		Mode:  mode,
		Atime: uint64(f.CreateTime.Unix()),
		Ctime: uint64(f.CreateTime.Unix()),
		Mtime: uint64(f.UpdateTime.Unix()),
		Owner: fuse.Owner{
			Uid: uint32(f.Uid),
			Gid: uint32(f.Gid),
		},
	}
}
