package fs

import (
	"errors"
	"fmt"
	"sync"
)

type Graph struct {
	nodes map[*File]*Node
	wait  sync.WaitGroup
}

type Node struct {
	file *File

	mutex           sync.RWMutex
	upstreamNodes   map[*Node]*Edge
	downstreamNodes map[*Node]*Edge
	queuedToRun     bool
	hasRun          bool
}

type Edge struct {
	from, to *Node
}

func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[*File]*Node),
	}
}

func (g *Graph) getNodeFor(file *File) *Node {
	node := g.nodes[file]
	if node == nil {
		node = &Node{
			file:            file,
			upstreamNodes:   make(map[*Node]*Edge),
			downstreamNodes: make(map[*Node]*Edge),
		}

		g.nodes[file] = node
	}

	return node
}

func (g *Graph) edge(from, to *Node) {
	edge := &Edge{
		from: from,
		to:   to,
	}

	from.downstreamNodes[to] = edge
	to.upstreamNodes[from] = edge
}

// Adds a file to the graph which acts as
func (g *Graph) AddTargetNode(file *File) error {
	visited := make(map[*File]struct{})

	g.addUpstreamModels(file, visited)

	// Check for circular dependencies & all nodes without upstreams
	for file := range visited {
		node := g.getNodeFor(file)

		if node.upstreamContains(node) {
			return errors.New(fmt.Sprintf("%s has a circular upstream dependency on itself", node.file.Name))
		}
	}

	return nil
}

func (g *Graph) addUpstreamModels(file *File, visited map[*File]struct{}) {
	if _, found := visited[file]; found {
		return
	}
	visited[file] = struct{}{}

	thisNode := g.getNodeFor(file)

	file.Mutex.Lock()
	defer file.Mutex.Unlock()
	for upstream := range file.upstreams {
		upstreamNode := g.getNodeFor(upstream)

		g.edge(upstreamNode, thisNode)

		g.addUpstreamModels(upstream, visited)
	}
}

func (g *Graph) Len() int {
	return len(g.nodes)
}

func (g *Graph) Execute(f func(file *File), numWorkers int) {
	c := make(chan *Node, len(g.nodes))

	g.wait.Add(len(g.nodes))

	worker := func() {
		for node := range c {
			f(node.file)

			node.markNodeAsRun(c)

			g.wait.Done()
		}
	}

	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	for _, nodes := range g.nodes {
		if nodes.allUpstreamsReady() {
			nodes.queueForRun(c)
		}
	}

	g.wait.Wait()
}

func (n *Node) upstreamContains(other *Node) bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for upstream := range n.upstreamNodes {
		if upstream == other {
			return true
		}

		if upstream.upstreamContains(other) {
			return true
		}
	}

	return false
}

func (n *Node) allUpstreamsReady() bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for upstream := range n.upstreamNodes {
		if !upstream.hasRun {
			return false
		}
	}

	return true
}

func (n *Node) queueForRun(c chan *Node) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if n.queuedToRun {
		return
	}

	n.queuedToRun = true
	c <- n
}

func (n *Node) markNodeAsRun(c chan *Node) {
	n.mutex.Lock()
	n.hasRun = true
	n.mutex.Unlock()

	n.mutex.RLock()
	defer n.mutex.RUnlock()

	// Now find any downstreams which are ready to run
	for downstream := range n.downstreamNodes {
		if downstream.allUpstreamsReady() {
			downstream.queueForRun(c)
		}
	}
}
