package examples

import "fmt"

type MockRepository struct {
	name string
}

func (m *MockRepository) Close() {
	fmt.Println("Close.")
}
