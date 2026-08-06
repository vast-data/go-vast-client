package common

import "testing"

func TestToUint64(t *testing.T) {
	v, err := ToUint64(42)
	if err != nil || v != 42 {
		t.Fatalf("ToUint64(42) = %d, %v", v, err)
	}
	if _, err := ToUint64(-1); err == nil {
		t.Fatal("expected error for negative value")
	}
}

func TestToUint(t *testing.T) {
	v, err := ToUint(7)
	if err != nil || v != 7 {
		t.Fatalf("ToUint(7) = %d, %v", v, err)
	}
	if _, err := ToUint(-1); err == nil {
		t.Fatal("expected error for negative value")
	}
}

func TestInt64FromUint(t *testing.T) {
	v, err := Int64FromUint(9)
	if err != nil || v != 9 {
		t.Fatalf("Int64FromUint(9) = %d, %v", v, err)
	}
}

func TestToUintFromUint64(t *testing.T) {
	v, err := ToUintFromUint64(5)
	if err != nil || v != 5 {
		t.Fatalf("ToUintFromUint64(5) = %d, %v", v, err)
	}
}

func TestUintFromFloat64(t *testing.T) {
	v, err := UintFromFloat64(3)
	if err != nil || v != 3 {
		t.Fatalf("UintFromFloat64(3) = %d, %v", v, err)
	}
	if _, err := UintFromFloat64(-1); err == nil {
		t.Fatal("expected error for negative float")
	}
}
