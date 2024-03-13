package node

import (
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func NewSQLFS(f func() sqlx.SqlConn) (fs.InodeEmbedder, error) {
	root := &FileNode{
		NewConn: f,
		PK:      0,
		Content: "",
	}
	root.root = root
	root.Parent = root

	return root, nil
}
