package main

import (
	"os"
	"testing"
)

func TestGetToken(t *testing.T) {
	token, err := getToken("alpine")
	if err != nil {
		t.Fatal(err)
	}

	if token == "" {
		t.Fatal(err)
	}
}

func TestGetLayers(t *testing.T) {
	token, err := getToken("alpine")
	if err != nil {
		t.Fatal(err)
	}

	layers, err := getLayers("alpine", "latest", token)
	if err != nil {
		t.Fatal(err)
	}

	if len(layers) == 0 {
		t.Fatal("layers is empty")
	}
}

func TestPullImage(t *testing.T) {
	dir, err := PullImage("alpine", "./images")
	if err != nil {
		t.Fatal(err)
	}

	stat, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size() == 0 {
		t.Fatal("image size is 0")
	}
}
