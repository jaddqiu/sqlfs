package node

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"syscall"
	"time"

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

	mu sync.Mutex
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
			}, fs.StableAttr{Ino: file.InodeId(), Mode: mode})
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
	out.Mode = 100755
	out.Size = uint64(len(fn.Content))
	return 0
}

func (fn *FileNode) Setattr(ctx context.Context, f fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)
	node, err := fm.FindOne(context.Background(), fn.PK)
	if err != nil {
		log.Println("find inode error: ", err)

		return syscall.EIO
	}

	if m, ok := in.GetMode(); ok {
		node.Mode = int64(m)
	}

	if uid, ok := in.GetUID(); ok {
		node.Uid = int64(uid)
	}

	if gid, ok := in.GetGID(); ok {
		node.Gid = int64(gid)
	}

	if mtime, ok := in.GetMTime(); ok {
		node.UpdateTime = mtime
	}

	err = fm.Update(context.Background(), node)
	if err != nil {
		log.Println("update file inode error: ", err)

		return syscall.EIO

	}

	out.Attr = node.Attr()

	return 0
}

func (fn *FileNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	fm := file.NewFileModel(fn.NewConn())
	_, err := fm.FindOneByParentDirName(context.Background(), fn.PK, name)
	switch err {
	case sql.ErrNoRows:
	default:
		log.Println("find inode error: ", err)

		errno = syscall.EIO
		return
	}
	child := &FileNode{
		NewConn: fn.NewConn,
		root:    fn.root,
		Parent:  fn,
	}

	f := &file.File{
		Name:       name,
		Type:       "file",
		ParentDir:  fn.PK,
		Mode:       int64(mode | fuse.S_IFREG),
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	r, err := fm.Insert(context.Background(), f)
	if err != nil {
		log.Println("insert file record error: ", err)
		errno = syscall.EIO

		return
	}

	child.PK, err = r.LastInsertId()
	if err != nil {
		log.Println("get last insert id error: ", err)
		errno = syscall.EIO
		return
	}

	out.NodeId = uint64(child.PK)
	out.Attr = f.Attr()
	node = fn.EmbeddedInode().NewInode(ctx, child, fs.StableAttr{Mode: mode | fuse.S_IFREG, Ino: f.InodeId(), Gen: 1})

	return
}

func (fn *FileNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)
	dentry, err := fm.FindOneByParentDirName(ctx, fn.PK, name)
	if err != nil {
		return &fs.Inode{}, syscall.ENOENT
	}
	attr := dentry.Attr()
	out.Attr = attr

	ch := fn.Inode.NewPersistentInode(ctx, &FileNode{
		root:    fn.root,
		PK:      dentry.Id,
		NewConn: fn.NewConn,
		Parent:  fn,
	},
		fs.StableAttr{Ino: dentry.InodeId(), Mode: attr.Mode},
	)
	return ch, 0

}

func (fn *FileNode) Write(ctx context.Context, fh fs.FileHandle, data []byte, off int64) (uint32, syscall.Errno) {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)
	node, err := fm.FindOne(context.Background(), fn.PK)
	if err != nil {
		log.Println("find inode error: ", err)

		return 0, syscall.EIO
	}
	end := int64(len(data)) + off

	content := []byte(node.Content.String)

	if int64(len(content)) < end {
		n := make([]byte, end)
		copy(n, content)
		content = n
	}

	copy(content[off:off+int64(len(data))], data)

	node.Content = sql.NullString{String: string(content), Valid: true}

	err = fm.Update(context.Background(), node)
	if err != nil {
		log.Println("update file inode error: ", err)

		return 0, syscall.EIO

	}

	fn.Content = string(content)

	return uint32(len(data)), 0
}

func (fn *FileNode) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	return 0
}

func (fn *FileNode) Fsync(ctx context.Context, f fs.FileHandle, flags uint32) syscall.Errno {
	return 0
}

func (fn *FileNode) Unlink(ctx context.Context, name string) syscall.Errno {

	fn.mu.Lock()
	defer fn.mu.Unlock()
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)

	node, err := fm.FindOneByParentDirName(context.Background(), fn.PK, name)
	if err != nil {
		log.Println("find inode error: ", err)

		return syscall.EIO
	}

	err = fm.Delete(context.Background(), node.Id)

	if err != nil {
		log.Println("delete file inode error: ", err)

		return syscall.EIO

	}
	return 0
}

func (fn *FileNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)

	node, err := fm.FindOneByParentDirName(context.Background(), fn.PK, name)
	if err != nil {
		log.Println("find inode error: ", err)

		return syscall.EIO
	}

	err = fm.Delete(context.Background(), node.Id)

	if err != nil {
		log.Println("delete file inode error: ", err)

		return syscall.EIO

	}
	return 0

}

func (fn *FileNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fn.mu.Lock()
	defer fn.mu.Unlock()
	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)

	_, err := fm.FindOneByParentDirName(ctx, fn.PK, name)
	switch err {
	case sql.ErrNoRows:
	default:
		log.Println("find inode error: ", err)

		return nil, syscall.EIO
	}

	child := &FileNode{
		NewConn: fn.NewConn,
		root:    fn.root,
		Parent:  fn,
	}

	f := &file.File{
		Name:       name,
		Type:       "directory",
		ParentDir:  fn.PK,
		Mode:       int64(mode | fuse.S_IFDIR),
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}
	r, err := fm.Insert(context.Background(), f)
	if err != nil {
		log.Println("insert file record error: ", err)
		return nil, syscall.EIO
	}

	child.PK, err = r.LastInsertId()
	if err != nil {
		log.Println("get last insert id error: ", err)
		return nil, syscall.EIO
	}

	out.NodeId = uint64(child.PK)
	out.Attr = f.Attr()
	node := fn.EmbeddedInode().NewInode(ctx, child, fs.StableAttr{Mode: mode | fuse.S_IFDIR, Ino: f.InodeId(), Gen: 1})
	return node, 0
}

func (fn *FileNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	fn.mu.Lock()
	defer fn.mu.Unlock()

	conn := fn.Root().NewConn()
	fm := file.NewFileModel(conn)

	f, err := fm.FindOneByParentDirName(context.Background(), fn.PK, name)
	switch err {
	case nil:
	case sql.ErrNoRows:
		log.Println("no such file: ", name)
		return syscall.EEXIST
	default:
		log.Println("find file error: ", err)

		return syscall.EIO
	}

	newParentFn := newParent.(*FileNode)
	targetFile, err := fm.FindOneByParentDirName(context.Background(), fn.PK, name)
	switch err {
	case nil:
		e := fm.Delete(context.Background(), targetFile.Id)
		if e != nil {
			log.Println("delete target file error")
		}
	case sql.ErrNoRows:
	default:
		log.Println("find target file error: ", err)

		return syscall.EIO
	}

	f.Name = newName
	f.ParentDir = newParentFn.PK

	err = fm.Update(context.Background(), f)
	if err != nil {
		log.Println("update node error: ", err)

		return syscall.EIO
	}

	return 0
}
