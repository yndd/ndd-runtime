module github.com/yndd/ndd-runtime

go 1.16

require (
	github.com/go-logr/logr v1.2.0
	github.com/openconfig/gnmi v0.0.0-20210914185457-51254b657b7d
	github.com/openconfig/goyang v1.0.0
	github.com/openconfig/ygot v0.15.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.6.0
	github.com/yndd/ndd-yang v0.2.0
	github.com/yndd/nddp-system v0.1.6
	github.com/yndd/registrator v0.0.19
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	k8s.io/api v0.23.6
	k8s.io/apimachinery v0.24.0
	k8s.io/client-go v0.23.5
	sigs.k8s.io/controller-runtime v0.11.2
)
