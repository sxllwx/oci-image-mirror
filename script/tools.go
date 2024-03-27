package script

import (
	_ "github.com/gogo/protobuf/gogoproto"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/policy/v1beta1"
	_ "k8s.io/apimachinery/pkg/api/resource"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	_ "k8s.io/apimachinery/pkg/apis/testapigroup/v1"
	_ "k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/code-generator"
)
