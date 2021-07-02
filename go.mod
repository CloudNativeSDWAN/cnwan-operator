module github.com/CloudNativeSDWAN/cnwan-operator

go 1.13

require (
	cloud.google.com/go v0.84.0
	github.com/aws/aws-sdk-go v1.38.60
	github.com/go-logr/logr v0.1.0
	github.com/googleapis/gax-go v1.0.3
	github.com/grpc-ecosystem/grpc-gateway v1.14.6 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	go.uber.org/zap v1.10.0
	google.golang.org/api v0.48.0
	google.golang.org/genproto v0.0.0-20210608205507-b6d2f5bf0d7d
	google.golang.org/grpc v1.38.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.25+incompatible
	go.etcd.io/bbolt => go.etcd.io/bbolt v1.3.5
	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20200910180754-dd1b699fc489 // ae9734ed278b is the SHA for git tag v3.4.13
	google.golang.org/grpc => google.golang.org/grpc v1.27.1
)
