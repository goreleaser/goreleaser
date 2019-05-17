// Package ids provides id validation code used my multiple pipes.
package ids

import "fmt"

// IDs is the IDs type
type IDs map[string]int

// New IDs
func New() IDs {
	return IDs(map[string]int{})
}

// Inc increment the counter of the given id
func (ids IDs) Inc(id string) {
	ids[id]++
}

// Validate errors if there are any ids with counter > 1
func (ids IDs) Validate() error {
	for id, cont := range ids {
		if cont > 1 {
			return fmt.Errorf("found %d items with the ID '%s', please fix your config", cont, id)
		}
	}
	return nil
}
