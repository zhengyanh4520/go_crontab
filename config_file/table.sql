CREATE TABLE user_table(
id varchar(20) primary key not null,
password varchar(20) not null,
name varchar(20) not null,
status int(2) not null default 1
);

CREATE TABLE task_table (
id varchar(50) primary key,
name varchar(50) not null default "",
user_id varchar(50) not null default "",
time_format varchar(50) not null default "",
command varchar(256) not null default "",
host varchar(50) not null default "",
timeout int not null default 0,
alone bool not null default false,
once bool not null default false,
repeats bool not null default false,
run_system bool not null default false,
share bool not null default false,
off bool not null default false,
index(host),
CONSTRAINT FOREIGN KEY (`user_id`) REFERENCES `user_table`(`id`)
ON DELETE CASCADE
ON UPDATE CASCADE
);

CREATE TABLE status_table (
id int primary key auto_increment not null,
host varchar(20) not null default "" comment '任务主机',
task_id varchar(50) not null default "",
start_time varchar(50) not null default '' comment '任务起始时间',
end_time varchar(50) not null default '' comment '任务结束时间',
status varchar(20) not null default "" comment '任务执行状态',
error varchar(100) not null default "",
insert_time  timestamp not null default CURRENT_TIMESTAMP,
index(task_id),
CONSTRAINT FOREIGN KEY (`task_id`) REFERENCES `task_table`(`id`)
ON DELETE CASCADE
ON UPDATE CASCADE
);

CREATE TABLE host_table(
name varchar(50) not null default "",
user_id varchar(20) not null,
host varchar(20) not null,
port varchar(20) not null,
PRIMARY KEY (`user_id`,`host`),
CONSTRAINT FOREIGN KEY (`user_id`) REFERENCES `user_table`(`id`)
ON DELETE CASCADE
ON UPDATE CASCADE
);