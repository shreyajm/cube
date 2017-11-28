package di

import (
	"github.com/twmb/algoimpl/go/graph"
	"fmt"
)

// DAG is responsible to gather all the components and their dependencies,
// and then returns a list of components sorted from no dependencies to
// the one with most dependencies. In that way we can prioritize creating the
// components and ensuring it is not going to fail because of lack of dependent object.

// Key for a component
type Key interface{}

// Value is the actual component object
type Value interface{}

// Graph is collection of vertices and edges between them.
// In dag, sort implement a topological sort for the graph.
type Graph interface {
	NewVertex(Key, Value) error
	Edge(Key, ...Key) error
	Sort() []Vertex
}

// NewDAG creates a new DAG.
func NewDAG() Graph {
	return &dag{
		graph.New(graph.Directed),
		make(map[Key]graph.Node, 0),
	}
}

// Vertex in the graph has the entry of <key, value> for a component
type Vertex struct {
	Key   Key
	Value Value
}

// dag is a Graph which has an internal graph.Graph inside itself handling
// nodes creation and making edges. It also has a map of all vertices which associates
// the key to a graph.Node that is holding their corresponding component.
type dag struct {
	graph    *graph.Graph
	vertices map[Key]graph.Node
}

func (dg *dag) NewVertex(key Key, val Value) error {
	if _, ok := dg.vertices[key]; ok {
		return fmt.Errorf("key %s already exists", key)
	}
	node := dg.graph.MakeNode()
	*node.Value = &Vertex{key, val}
	dg.vertices[key] = node
	return nil
}


func (dg *dag) Edge(node Key, dependencies ...Key) error {
	for _, dep := range dependencies {
		if err := dg.addEdge(node, dep); err != nil {
			return err
		}
	}
	return nil
}

// Edge creates a dependency between two vertices. This returns an error
// if either the node or the dependency is not present in the graph or
// a make a node depends on itself. Adding an edge that creates
// a cycle in the graph is not allowed.
func (dg *dag) addEdge(node Key, dependency Key) error {
	srcObj, ok := dg.vertices[node];
	if !ok {
		return fmt.Errorf("key %s does not exist", node)
	}
	dstObj, ok := dg.vertices[dependency]
	if !ok {
		return fmt.Errorf("key %s does not exist", dependency)
	}

	if node == dependency {
		return fmt.Errorf("edge to self is not allowed")
	}

	dg.graph.MakeEdge(dstObj, srcObj)

	if !isAcyclic(dg) {
		dg.graph.RemoveEdge(dstObj, srcObj)
		return fmt.Errorf("edge from %s to %s makes a cycle", node, dependency)
	}
	return nil

}

// A sorted traversal of this graph will guarantee the
// dependency order. This means A (node) depends on B (dependency) then
// the sorted traversal will always return B before A.
func (dg *dag) Sort() []Vertex {
	sorted := dg.graph.TopologicalSort()
	nodes := make([]Vertex, 0, len(sorted))
	for _, n := range sorted {
		vp := (*n.Value).(*Vertex)
		n := Vertex{vp.Key, vp.Value}
		nodes = append(nodes, n)
	}
	return nodes
}

// Determines whether a DAG is acyclic or not.
func isAcyclic(dg *dag) bool {
	connectedComponents := dg.graph.StronglyConnectedComponents()
	// If the arrays underlying each node has a size of one, it means that each
	// vertex in the dag is a connected component. There's not any connected component
	// with more than one vertex in it. Therefore there isn't any cycle in the DAG.
	for _, arr := range connectedComponents {
		if len(arr) > 1 {
			return false
		}
	}
	return true
}
