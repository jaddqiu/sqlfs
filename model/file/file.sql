create table file(
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT 'primary key',
  `name` varchar(128) NOT NULL DEFAULT '' COMMENT '文件名',
  `type` enum('file','directory') DEFAULT 'file' COMMENT '文件类型',
  `parent_dir` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父目录id',
  `content` text COMMENT '文件内容',
  `uid` int unsigned not NULL default '0',
  `gid` int unsigned not NULL default '0',
  `mode` int unsigned not NULL default '0',
  
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_parent_dir` (`parent_dir`),
  Unique key `idx_parent_name` (`parent_dir`, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
