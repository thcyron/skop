package intstr

import "testing"

func TestInt32(t *testing.T) {
	x := Int32(42)
	if x.Type == nil || *x.Type != 0 {
		t.Errorf("unexpected type: %v", x.Type)
	}
	if x.IntVal == nil || *x.IntVal != 42 {
		t.Errorf("unexpected integer value: %v", x.IntVal)
	}
	if x.StrVal != nil {
		t.Errorf("unexpected string value: %v", x.StrVal)
	}
}

func TestString(t *testing.T) {
	x := String("test")
	if x.Type == nil || *x.Type != 1 {
		t.Errorf("unexpected type: %v", x.Type)
	}
	if x.IntVal != nil {
		t.Errorf("unexpected integer value: %v", x.IntVal)
	}
	if x.StrVal == nil || *x.StrVal != "test" {
		t.Errorf("unexpected string value: %v", x.StrVal)
	}
}
