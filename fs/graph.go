package fs

import (
	"fmt"
	"sync"

	"ddbt/utils"
)

type Graph struct {
	nodes map[*File]*Node
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
			return fmt.Errorf("%s has a circular upstream dependency on itself", node.file.Name)
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
			return fmt.Errorf("%s has a circular dependency on itself", node.file.Name)
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
			return fmt.Errorf("%s has a circular dependency on itself", node.file.Name)
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
			return fmt.Errorf("%s has a circular dependency on itself", node.file.Name)
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

func (g *Graph) AddAllUsedMacros() error {
	visited := make(map[*File]struct{})

	for file := range g.nodes {
		g.addUpstreamMacros(file, visited)
	}

	// Check for circular dependencies & all nodes without upstreams
	for file := range visited {
		node := g.getNodeFor(file)

		if node.upstreamContains(node) {
			return fmt.Errorf("%s has a circular dependency on itself", node.file.Name)
		}
	}

	return nil
}

func (g *Graph) addUpstreamMacros(file *File, visited map[*File]struct{}) {
	if _, found := visited[file]; found {
		return
	}
	visited[file] = struct{}{}

	thisNode := g.getNodeFor(file)

	file.Mutex.Lock()
	defer file.Mutex.Unlock()
	for upstream := range file.upstreams {
		if upstream.Type == MacroFile {
			upstreamNode := g.getNodeFor(upstream)

			g.edge(upstreamNode, thisNode)

			g.addUpstreamMacros(upstream, visited)
		}
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

// addEphemeralUpstreamsFor recursively adds upstream ephemeral nodes to a given node
func (g *Graph) addEphemeralUpstreamsFor(node *Node) {
	for upstream := range node.file.upstreams {
		if upstream.Type != ModelFile {
			continue
		}

		if upstream.GetMaterialization() != "ephemeral" {
			continue
		}

		upstreamNode := g.getNodeFor(upstream)
		g.edge(upstreamNode, node)
		g.addEphemeralUpstreamsFor(upstreamNode)
	}
}

// AddEphemeralUpstreams brings any upstream models that are ephemeral into the graph
func (g *Graph) AddEphemeralUpstreams() {
	for _, node := range g.nodes {
		if node.file.Type != ModelFile {
			continue
		}
		g.addEphemeralUpstreamsFor(node)
	}
}

func (g *Graph) Len() int {
	return len(g.nodes)
}

func (g *Graph) ListNodes() map[*File]*Node {
	return g.nodes
}

func (g *Graph) Execute(f func(file *File) error, numWorkers int, pb *utils.ProgressBar) error {
	var wait sync.WaitGroup

	countOfUnqueued := g.NumberNodesNeedRerunning()
	c := make(chan *Node, countOfUnqueued)

	wait.Add(countOfUnqueued)
	end := make(chan struct{})

	var errMutex sync.RWMutex
	var firstErr error

	worker := func() {
		statusRow := pb.NewStatusRow()

		for node := range c {
			errMutex.RLock()
			if firstErr != nil {
				errMutex.RUnlock()
				return
			}
			errMutex.RUnlock()

			statusRow.Update(fmt.Sprintf("Running %s", node.file.Name))

			err := f(node.file)
			if err != nil {
				errMutex.Lock()
				if firstErr == nil {
					firstErr = err
					end <- struct{}{}
				}
				errMutex.Unlock()

				return
			}
			node.markNodeAsRun(c)
			wait.Done()

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

	// Wait on the
	go func() {
		wait.Wait()
		end <- struct{}{}
	}()
	<-end

	close(c)

	errMutex.RLock()
	defer errMutex.RUnlock()

	return firstErr
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

	// Now find any downstreams which are ready to run
	if c != nil {
		n.mutex.RLock()
		defer n.mutex.RUnlock()

		for downstream := range n.downstreamNodes {
			if downstream.allUpstreamsReady() {
				downstream.queueForRun(c)
			}
		}
	}
}

func (g *Graph) MarkGraphAsFullyRun() {
	for _, node := range g.nodes {
		node.queuedToRun = true
		node.hasRun = true
	}
}

func (g *Graph) UnmarkGraphAsFullyRun() {
	for _, node := range g.nodes {
		node.queuedToRun = false
		node.hasRun = false
	}
}

// If a file is in the graph, this removes it's queuedToRun and hasRun flags
func (g *Graph) UnmarkFileAsRun(file *File) {
	node, found := g.nodes[file]
	if found {
		node.queuedToRun = false
		node.hasRun = false
	}
}

func (g *Graph) NumberNodesNeedRerunning() int {
	count := 0
	for _, node := range g.nodes {
		if node.hasRun == false {
			count++
		}
	}

	return count
}

func (g *Graph) Contains(file *File) bool {
	_, found := g.nodes[file]
	return found
}
