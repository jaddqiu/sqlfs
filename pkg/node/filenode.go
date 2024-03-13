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
	Parent  *FileNode
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

		if file.Type != "directory" {
			return
		}
	}
	files, err := fm.GetChildrenByParentId(ctx, fn.PK)
	if err != nil {
		log.Println("get children error: ", err)
		return
	}
	for _, file := range files {
		ch := fn.GetChild(file.Name)
		if ch == nil {
			var mode uint32
			if file.Type == "directory" {
				mode = fuse.S_IFDIR
			} else {
				mode = fuse.S_IFREG
			}
			ch := fn.NewPersistentInode(ctx, &FileNode{
				root:    fn.root,
				PK:      file.Id,
				NewConn: fn.NewConn,
				Content: file.Content.String,
				Parent:  fn,
			}, fs.StableAttr{Ino: uint64(file.Id + 10000), Mode: mode})
			fn.AddChild(file.Name, ch, false)
		}
	}

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
	return readdir(ctx, fn)
}

func (fn *FileNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = 0755
	out.Size = uint64(len(fn.Content))
	return 0
}

func (fn *FileNode) Lookup1(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)
	dentry, err := fm.FindOneByParentDirName(ctx, fn.PK, name)
	if err != nil {
		return &fs.Inode{}, syscall.ENOENT
	}
	st := syscall.Stat_t{}
	if dentry.Type == "directory" {
		st.Mode = fuse.S_IFDIR
	} else {
		st.Mode = fuse.S_IFREG
		st.Size = int64(len(dentry.Content.String))
	}

	out.Attr.FromStat(&st)

	ch := fn.Inode.NewPersistentInode(ctx, &FileNode{
		root:    fn.root,
		PK:      dentry.Id,
		NewConn: fn.NewConn,
		Parent:  fn,
	},
		fs.StableAttr{Ino: uint64(dentry.Id + 10000), Mode: st.Mode},
	)
	return ch, 0

}
