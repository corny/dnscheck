package main

import "testing"

func TestCompare(t *testing.T) {
  result, err := resolve("8.8.8.8", "example.com")

func TestExistent(t *testing.T) {
	result, err := resolve("8.8.8.8", "example.com")

	if err != nil {
		t.Fatal("an error occured")
	}

	if len(result) != 1 {
		t.Fatal("invalid number of records returned:", len(result))
	}
}

func TestNotExistent(t *testing.T) {
	result, err := resolve("8.8.8.8", "xxx.example.com")

	if err != nil {
		t.Fatal("an error occured")
	}

	if len(result) > 0 {
		t.Fatal("records returned")
	}
}
