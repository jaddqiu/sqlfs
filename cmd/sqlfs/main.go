package main

import (
	"flag"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"sql-fs/pkg/conf"
	"sql-fs/pkg/mysql"
	"sql-fs/pkg/node"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func main() {
	debug := flag.Bool("debug", false, "print debugging messages.")
	mount := flag.String("mount", "", "conf file path")
	conf_file := flag.String("config", "", "conf file path")
	writePprof := flag.Bool("pprof", false, "write cpu and mem pprof file")

	flag.Parse()

	var err error
	conf.Conf, err = conf.LoadConfg(*conf_file)
	if err != nil {
		log.Fatalf("load conf error: %v, conf path: %s", err, *conf_file)
	}

	profile := conf.Conf.Pprof.CPUPprofFile
	mem_profile := conf.Conf.Pprof.MemPprofFile

	var profFile, memProfFile io.Writer

	if *writePprof {
		profFile, err = os.Create(profile)
		if err != nil {
			log.Fatalf("os.Create: %v", err)
		}
		memProfFile, err = os.Create(mem_profile)
		if err != nil {
			log.Fatalf("os.Create: %v", err)
		}

		pprof.StartCPUProfile(profFile)
		defer pprof.StopCPUProfile()
	}

	opts := &fs.Options{
		AttrTimeout:  &conf.Conf.Fuse.TTL,
		EntryTimeout: &conf.Conf.Fuse.TTL,
	}
	opts.Debug = *debug

	f := func() sqlx.SqlConn {
		return mysql.New(
			conf.Conf.MySQL.Host,
			conf.Conf.MySQL.Port,
			conf.Conf.MySQL.User,
			conf.Conf.MySQL.Password,
			conf.Conf.MySQL.DB,
		)
	}

	var root fs.InodeEmbedder
	root, err = node.NewSQLFS(f)
	if err != nil {
		log.Fatalf("os.Create: %v", err)
	}

	server, err := fs.Mount(*mount, root, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}

	server.Wait()

	if *writePprof {
		pprof.WriteHeapProfile(memProfFile)
	}

}
