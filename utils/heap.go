package utils

import "distribuidos/tp1/server/middleware"

type GameHeap []middleware.AvgPlaytimeGame

func (g GameHeap) Len() int { return len(g) }
func (g GameHeap) Less(i, j int) bool {
	return g[i].AveragePlaytimeForever < g[j].AveragePlaytimeForever
}
func (g GameHeap) Swap(i, j int) { g[i], g[j] = g[j], g[i] }
func (g *GameHeap) Push(x any) {
	*g = append(*g, x.(middleware.AvgPlaytimeGame))
}

func (g *GameHeap) Pop() any {
	old := *g
	n := len(old)
	x := old[n-1]
	*g = old[0 : n-1]
	return x
}

func (g *GameHeap) Peek() any {
	return (*g)[0]
}