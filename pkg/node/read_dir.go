package node

import (
	"context"

	"sql-fs/model/file"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type FileStream struct {
	files []*file.File
	index int
}

func (ds *FileStream) Next() (fuse.DirEntry, syscall.Errno) {
	if ds.index >= len(ds.files) {
		return fuse.DirEntry{}, syscall.Errno(fuse.OK)
	}
	f := ds.files[ds.index]
	ds.index++
	var mode uint32
	if f.Type == "directory" {
		mode = fuse.S_IFDIR
	} else {
		mode = fuse.S_IFREG
	}

	return fuse.DirEntry{
		Name: f.Name,
		Ino:  uint64(f.Id + 10000),
		Mode: mode,
	}, syscall.Errno(fuse.OK)
}

func (ds *FileStream) Close() {
}

func (ds *FileStream) HasNext() bool {
	return ds.index < len(ds.files)
}

// readdir: return file list in the directory
func readdir(ctx context.Context, dir *FileNode) (fs.DirStream, syscall.Errno) {
	defaultFiles := []*file.File{
		{
			Id:   dir.PK,
			Name: ".",
			Type: "directory",
		},
		{
			Id:   dir.Parent.PK,
			Name: "..",
			Type: "directory",
		},
	}

	fm := file.NewFileModel(dir.NewConn())
	files, err := fm.GetChildrenByParentId(ctx, dir.PK)
	if err != nil {
		return nil, syscall.EIO
	}
	files = append(defaultFiles, files...)
	return &FileStream{
		files: files,
		index: 0,
	}, 0
}
