package metal

import "fmt"

var (
	ErrNotFound = fmt.Errorf("not found")
)

func IsNotFound(e error) bool {
	return e == ErrNotFound
}
