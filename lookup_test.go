package main

import "testing"

func TestExistent(t *testing.T) {
	result, err := resolve(referenceNameserver, "example.com")

	if err != nil {
		t.Fatal("an error occured")
	}

	if len(result) != 1 {
		t.Fatal("invalid number of records returned:", len(result))
	}
}

func TestNotExistent(t *testing.T) {
	result, err := resolve(referenceNameserver, "xxx.example.com")

	if err != nil {
		t.Fatal("an error occured")
	}

	if len(result) > 0 {
		t.Fatal("no records expected")
	}
}

func TestUnreachable(t *testing.T) {
	_, err := resolve("127.1.2.3", "example.com")

	if err == nil {
		t.Fatal("no error returned")
	}
	if err.Error() != "connection refused" {
		t.Fatal("unexpected error", err)
	}
}

func TestPtrName(t *testing.T) {
	result := ptrName("8.8.8.8")

	if result != "google-public-dns-a.google.com." {
		t.Fatal("invalid result:", result)
	}
}
