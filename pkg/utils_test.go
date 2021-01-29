package pkg

import (
	"fmt"
	"os/user"
	"path/filepath"
	"testing"
)

func TestExpandUser(t *testing.T) {

	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	homeDir := currentUser.HomeDir

	expectedPath := filepath.Join(homeDir, ".config/bleachbit")

	if ExpandUser("~/.config/bleachbit") != expectedPath {
		t.Fatal("wrong")
	}
}

func TestReadPasswordFile(t *testing.T) {
	mapIntPwd, mapStringPwd, err := readPasswordFile()
	if err != nil || err == errNoPassDB {
		t.Fatalf("Failed: %v", err)
	}

	fmt.Println(mapIntPwd)
	fmt.Println(mapStringPwd)
}
