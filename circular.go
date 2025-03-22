package gocfg

import (
	"fmt"

	"github.com/Gardego5/gocfg/utils"
)

type node struct {
	fieldName    string
	fieldIndex   int
	tag          string
	loader       Loader
	dependencies []string
	resolved     bool
}

// detectCircularDependencies checks for circular dependencies in the graph
func detectCircularDependencies(nodes map[string]*node) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var checkCycle func(nodeName string) error
	checkCycle = func(nodeName string) error {
		node, exists := nodes[nodeName]
		if !exists {
			return fmt.Errorf("%w: %s", utils.ErrUnboundVariable, nodeName)
		}

		visited[nodeName] = true
		recStack[nodeName] = true

		for _, dep := range node.dependencies {
			if !visited[dep] {
				if err := checkCycle(dep); err != nil {
					return err
				}
			} else if recStack[dep] {
				return fmt.Errorf("%w: %s -> %s", utils.ErrCircularDependency, nodeName, dep)
			}
		}

		recStack[nodeName] = false
		return nil
	}

	for nodeName := range nodes {
		if !visited[nodeName] {
			if err := checkCycle(nodeName); err != nil {
				return err
			}
		}
	}

	return nil
}
