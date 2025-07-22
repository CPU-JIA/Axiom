module cicd-service

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.3.1
	github.com/spf13/viper v1.17.0
	github.com/golang-jwt/jwt/v5 v5.2.0
	gorm.io/driver/postgres v1.5.3
	gorm.io/gorm v1.25.5
	gorm.io/datatypes v1.2.0
	k8s.io/api v0.28.4
	k8s.io/apimachinery v0.28.4
	k8s.io/client-go v0.28.4
	github.com/tektoncd/pipeline v0.53.0
	github.com/gorilla/websocket v1.5.0
	github.com/robfig/cron/v3 v3.0.1
	gopkg.in/yaml.v3 v3.0.1
)