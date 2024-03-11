package node

import (
	"context"
	"log"
	"syscall"

	"sql-fs/model/file"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type FileNode struct {
	fs.Inode
	root    *FileNode
	PK      int64
	Content string
	NewConn func() sqlx.SqlConn
}

func (fn *FileNode) Root() *FileNode {
	return fn.root
}

func (fn *FileNode) OnAdd(ctx context.Context) {
	fm := file.NewFileModel(fn.NewConn())
	if fn.PK != 0 {
		file, err := fm.FindOne(ctx, fn.PK)
		if err != nil {
			log.Printf("find node error: %v\n", err)
			return
		}

		if file.Type != "dirctory" {
			return
		}
	}

	files, errno := readdir(ctx, fn.root, fn.PK)
	if errno != 0 {
		log.Printf("mode add error")
		return
	}
	node := fn.EmbeddedInode()
	for files.HasNext() {
		entry, errno := files.Next()
		if errno != 0 {
			log.Printf("files next error")
		}
		ch := node.GetChild(entry.Name)
		if ch == nil {
			ch = node.NewPersistentInode(ctx, &FileNode{root: fn.root, PK: int64(entry.Ino - 10000), NewConn: fn.NewConn}, fs.StableAttr{Ino: entry.Ino, Mode: entry.Mode})
			node.AddChild(entry.Name, ch, false)
		}
	}
	files.Close()

}

func (fn *FileNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)
	node, err := fm.FindOne(context.Background(), fn.PK)
	if err != nil {
		return nil, 0, syscall.EIO
	}

	if node.Type == "file" {
		fn.Content = node.Content.String
	}

	return nil, 0, 0
}

func (fn *FileNode) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	end := int(off) + len(dest)
	if end > len(fn.Content) {
		end = len(fn.Content)
	}
	return fuse.ReadResultData([]byte(fn.Content)[off:end]), 0
}

func (fn *FileNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	return readdir(ctx, fn.root, fn.PK)
}

func (fn *FileNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	return 0
}
