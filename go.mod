module go_crontab

go 1.15

require (
	github.com/coreos/etcd v3.3.25+incompatible
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/gin-gonic/gin v1.7.1
	github.com/go-sql-driver/mysql v1.6.0
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.2.0
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5
	github.com/robfig/cron/v3 v3.0.1
	github.com/sirupsen/logrus v1.8.1
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/text v0.3.6
	google.golang.org/genproto v0.0.0-20210427215850-f767ed18ee4d // indirect
	google.golang.org/grpc v1.37.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0