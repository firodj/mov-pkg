package exampletwos

import (
	"github.com/firodj/mov-pkg/examples"
)

func Freeze() *examples.DServiceCategory {
	return &examples.DServiceCategory{
		ID: 20,
		DServiceType: &examples.DServiceType{
			Username: "Nidu",
		},
	}
}
