package pkg

import (
	"fmt"
)

type Delete struct {
	path  string
	shred bool
}

func NewDelete(path string) *Delete {
	return &Delete{
		path,
		false,
	}
}

// String implements fmt's Stringer interface
func (d *Delete) String() string {
	shr := "shred"
	if !d.shred {
		shr = "delete"
	}
	return fmt.Sprintf("Command to %s %s", shr, d.path)
}

func (d *Delete) execute() {

}
