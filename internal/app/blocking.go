package app

import (
	"strings"

	"lazybeads/internal/models"
)

func applyBlockingDepth(tasks []models.Task) {
	index := make(map[string]*models.Task, len(tasks))
	for i := range tasks {
		tasks[i].BlockingDepth = 0
		index[tasks[i].ID] = &tasks[i]
	}

	memo := make(map[string]int, len(tasks))
	visiting := make(map[string]bool, len(tasks))

	var depthFor func(id string) int
	depthFor = func(id string) int {
		if depth, ok := memo[id]; ok {
			return depth
		}

		task, ok := index[id]
		if !ok || len(task.BlockedBy) == 0 {
			memo[id] = 0
			return 0
		}

		if visiting[id] {
			return 0
		}
		visiting[id] = true

		maxDepth := 0
		for _, blocker := range task.BlockedBy {
			depth := 1
			if _, ok := index[blocker]; ok {
				depth += depthFor(blocker)
			}
			if depth > maxDepth {
				maxDepth = depth
			}
		}

		visiting[id] = false
		if maxDepth == 0 {
			maxDepth = 1
		}

		memo[id] = maxDepth
		return maxDepth
	}

	for i := range tasks {
		tasks[i].BlockingDepth = depthFor(tasks[i].ID)
	}
}

func orderTasksByBlockingTree(tasks []models.Task) []models.Task {
	if len(tasks) == 0 {
		return tasks
	}

	indexByID := make(map[string]int, len(tasks))
	for i := range tasks {
		indexByID[tasks[i].ID] = i
		tasks[i].TreePrefix = ""
	}

	parentByID := make(map[string]string, len(tasks))
	for i := range tasks {
		task := tasks[i]
		for _, blocker := range task.BlockedBy {
			if _, ok := indexByID[blocker]; ok {
				parentByID[task.ID] = blocker
				break
			}
		}
	}

	childrenByID := make(map[string][]string, len(tasks))
	for i := range tasks {
		id := tasks[i].ID
		if parent, ok := parentByID[id]; ok && parent != "" {
			childrenByID[parent] = append(childrenByID[parent], id)
		}
	}

	var roots []string
	for i := range tasks {
		id := tasks[i].ID
		if parentByID[id] == "" {
			roots = append(roots, id)
		}
	}
	if len(roots) == 0 {
		for i := range tasks {
			roots = append(roots, tasks[i].ID)
		}
	}

	visited := make(map[string]bool, len(tasks))
	ordered := make([]models.Task, 0, len(tasks))

	var dfs func(id string, ancestors []bool, hasNext bool, depth int)
	dfs = func(id string, ancestors []bool, hasNext bool, depth int) {
		if visited[id] {
			return
		}
		visited[id] = true

		task := tasks[indexByID[id]]
		if depth > 0 {
			task.TreePrefix = buildTreePrefix(ancestors, hasNext)
		} else {
			task.TreePrefix = ""
		}
		ordered = append(ordered, task)

		children := childrenByID[id]
		for i, childID := range children {
			childHasNext := i < len(children)-1
			nextAncestors := append(ancestors, hasNext)
			dfs(childID, nextAncestors, childHasNext, depth+1)
		}
	}

	for i, rootID := range roots {
		rootHasNext := i < len(roots)-1
		dfs(rootID, nil, rootHasNext, 0)
	}

	if len(ordered) < len(tasks) {
		for i := range tasks {
			id := tasks[i].ID
			if visited[id] {
				continue
			}
			task := tasks[i]
			task.TreePrefix = ""
			ordered = append(ordered, task)
		}
	}

	return ordered
}

func buildTreePrefix(ancestors []bool, hasNext bool) string {
	var b strings.Builder
	for _, ancestorHasNext := range ancestors {
		if ancestorHasNext {
			b.WriteString("│  ")
		} else {
			b.WriteString("   ")
		}
	}
	if hasNext {
		b.WriteString("├─ ")
	} else {
		b.WriteString("└─ ")
	}
	return b.String()
}
