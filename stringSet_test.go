package main

import "testing"

func TestCompare(t *testing.T) {
	set1 := make(stringSet)
	set1.add("bar")
	set1.add("foo")

	set2 := make(stringSet)
	set2.add("foo")
	set2.add("bar")

	set3 := make(stringSet)
	set3.add("foo")
	set3.add("baz")

	if set1.equals(set2) != true {
		t.Fatal("set1 and set2 should be equal")
	}

	if set2.equals(set1) != true {
		t.Fatal("set2 and set1 should be equal")
	}

	if set2.equals(set3) == true {
		t.Fatal("set2 and set1 should NOT be equal")
	}
}

func TestString(t *testing.T) {
	set := make(stringSet)
	set.add("bar")
	set.add("xx")
	set.add("foo")

	str := set.String()

	if str != "bar, foo, xx" {
		t.Fatal("unexpected result:", str)
	}
}
