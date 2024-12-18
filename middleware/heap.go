package middleware

type GameHeap []GameStat

func (g GameHeap) Len() int { return len(g) }
func (g GameHeap) Less(i, j int) bool {
	return g[i].Stat < g[j].Stat
}
func (g GameHeap) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g *GameHeap) Push(x any) {
	*g = append(*g, x.(GameStat))
}

func (g *GameHeap) Pop() any {
	old := *g
	n := len(old)
	x := old[n-1]
	*g = old[0 : n-1]
	return x
}

func (g *GameHeap) Peek() GameStat {
	return (*g)[0]
}
