package fs

import (
	"errors"
	"fmt"
	"sync"

	"ddbt/utils"
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

		file.MarkAsInDAG()
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

func (g *Graph) AddAllModels(fs *FileSystem) error {
	visited := make(map[*File]struct{})

	for _, file := range fs.files {
		if file.Type == ModelFile {
			g.addUpstreamModels(file, visited)
		}
	}

	// Check for circular dependencies & all nodes without upstreams
	for file := range visited {
		node := g.getNodeFor(file)

		if node.upstreamContains(node) {
			return errors.New(fmt.Sprintf("%s has a circular upstream dependency on itself", node.file.Name))
		}
	}

	return nil
}

func (g *Graph) AddNode(file *File) {
	g.getNodeFor(file)
}

func (g *Graph) addNodeAndUpstreamsWithTag(file *File, tag string, visited map[*File]struct{}) {
	if _, found := visited[file]; found {
		return
	}
	visited[file] = struct{}{}

	node := g.getNodeFor(file)

	for upstream := range file.upstreams {
		if upstream.HasTag(tag) {
			upstreamNode := g.getNodeFor(upstream)
			g.edge(upstreamNode, node)

			g.addNodeAndUpstreamsWithTag(file, tag, visited)
		}
	}
}

func (g *Graph) AddFilesWithTag(fs *FileSystem, tag string) error {
	visited := make(map[*File]struct{})

	for _, file := range fs.AllFiles() {
		if file.HasTag(tag) {
			g.addNodeAndUpstreamsWithTag(file, tag, visited)
		}
	}

	// Check for circular dependencies & all nodes without upstreams
	for file := range visited {
		node := g.getNodeFor(file)

		if node.upstreamContains(node) {
			return errors.New(fmt.Sprintf("%s has a circular dependency on itself", node.file.Name))
		}
	}

	return nil
}

// Adds a file to the graph which acts as
func (g *Graph) AddNodeAndUpstreams(file *File) error {
	visited := make(map[*File]struct{})

	g.addUpstreamModels(file, visited)

	// Check for circular dependencies & all nodes without upstreams
	for file := range visited {
		node := g.getNodeFor(file)

		if node.upstreamContains(node) {
			return errors.New(fmt.Sprintf("%s has a circular dependency on itself", node.file.Name))
		}
	}

	return nil
}

// Adds a file to the graph which acts as
func (g *Graph) AddNodeAndDownstreams(file *File) error {
	visited := make(map[*File]struct{})

	g.addDownstreamModels(file, visited)

	// Check for circular dependencies & all nodes without upstreams
	for file := range visited {
		node := g.getNodeFor(file)

		if node.downstreamContains(node) {
			return errors.New(fmt.Sprintf("%s has a circular dependency on itself", node.file.Name))
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

func (g *Graph) addDownstreamModels(file *File, visited map[*File]struct{}) {
	if _, found := visited[file]; found {
		return
	}
	visited[file] = struct{}{}

	thisNode := g.getNodeFor(file)

	file.Mutex.Lock()
	defer file.Mutex.Unlock()
	for downstream := range file.downstreams {
		downstreamNode := g.getNodeFor(downstream)

		g.edge(thisNode, downstreamNode)

		g.addDownstreamModels(downstream, visited)
	}
}

// Find all tests which reference the models in the existing graph
// and add them to the graph
//
// returns the tests
func (g *Graph) AddReferencingTests() []*File {
	foundTest := make(map[*File]struct{})

	for _, node := range g.nodes {
		if node.file.Type != ModelFile {
			continue
		}

		for downstream := range node.file.downstreams {
			if downstream.Type != TestFile {
				continue
			}

			downstreamNode := g.getNodeFor(downstream)
			g.edge(node, downstreamNode)
			foundTest[downstream] = struct{}{}
		}
	}

	tests := make([]*File, 0, len(foundTest))
	for test := range foundTest {
		tests = append(tests, test)
	}

	return tests
}

func (g *Graph) Len() int {
	return len(g.nodes)
}

func (g *Graph) Execute(f func(file *File), numWorkers int, pb *utils.ProgressBar) {
	c := make(chan *Node, len(g.nodes))

	g.wait.Add(len(g.nodes))

	worker := func() {
		statusRow := pb.NewStatusRow()

		for node := range c {
			statusRow.Update(fmt.Sprintf("Running %s", node.file.Name))

			f(node.file)
			node.markNodeAsRun(c)
			g.wait.Done()

			statusRow.SetIdle()
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

func (n *Node) downstreamContains(other *Node) bool {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	for downstream := range n.downstreamNodes {
		if downstream == other {
			return true
		}

		if downstream.downstreamContains(other) {
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
