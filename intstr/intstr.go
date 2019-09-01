package intstr

import "github.com/ericchiang/k8s/util/intstr"

func Int32(i int32) *intstr.IntOrString {
	typ := int64(0)
	return &intstr.IntOrString{Type: &typ, IntVal: &i}
}

func String(s string) *intstr.IntOrString {
	typ := int64(1)
	return &intstr.IntOrString{Type: &typ, StrVal: &s}
}
