// Package check wires the pipeline together: load -> diff -> classify.
// The CLI and the tests both call this, so they exercise identical logic.
package check

import (
	"guardian/internal/diff"
	"guardian/internal/loader"
	"guardian/internal/rules"
)

func Run(basePath, prPath string) ([]diff.Change, error) {
	base, err := loader.Load(basePath)
	if err != nil {
		return nil, err
	}
	pr, err := loader.Load(prPath)
	if err != nil {
		return nil, err
	}
	return rules.Classify(diff.Compare(base, pr)), nil
}
