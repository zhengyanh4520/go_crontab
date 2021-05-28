CREATE TABLE UserTable(
id varchar(20) primary key not null comment '用户id',
password varchar(20) not null comment '用户密码',
name varchar(20) not null comment '用户名称',
company varchar(20) not null comment '工作单位',
department varchar(20) not null comment '所属部门',
duties varchar(20) not null comment '担任职务',
phone varchar(12) not null comment '手机号码'
);

CREATE TABLE TaskTable (
id varchar(50) primary key comment '任务id',
name varchar(50) not null default "" comment '任务名称',
user_id varchar(20) not null default "" comment '所属用户',
time_format varchar(50) not null default "" comment '时间语法',
command varchar(256) not null default "" comment '执行命令',
host varchar(20) not null default "" comment '执行主机',
timeout int not null default 0 comment '超时时间',
alone bool not null default false comment '单机属性',
once bool not null default false comment '定时属性',
repeats bool not null default false comment '重复属性',
run_system bool not null default false comment '主机系统',
share bool not null default false comment '共享属性',
off bool not null default false comment '开启/关闭',
index(host),
CONSTRAINT FOREIGN KEY (`user_id`) REFERENCES `UserTable`(`id`)
ON DELETE CASCADE
ON UPDATE CASCADE
);

CREATE TABLE StatusTable (
id int primary key auto_increment not null,
host varchar(20) not null default "" comment '任务主机',
task_id varchar(50) not null default "" comment '任务id',
start_time varchar(20) not null default '' comment '任务起始时间',
end_time varchar(20) not null default '' comment '任务结束时间',
status varchar(10) not null default "" comment '任务执行状态',
error varchar(100) not null default "" comment '错误原因',
insert_time  timestamp not null default CURRENT_TIMESTAMP,
index(task_id),
CONSTRAINT FOREIGN KEY (`task_id`) REFERENCES `TaskTable`(`id`)
ON DELETE CASCADE
ON UPDATE CASCADE
);

CREATE TABLE HostTable(
name varchar(50) not null default "" comment '主机名称',
user_id varchar(20) not null comment '所属用户',
host varchar(20) not null comment '主机号',
port varchar(10) not null comment '端口号',
PRIMARY KEY (`user_id`,`host`),
CONSTRAINT FOREIGN KEY (`user_id`) REFERENCES `UserTable`(`id`)
ON DELETE CASCADE
ON UPDATE CASCADE
);